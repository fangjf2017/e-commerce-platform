package com.ecommerce.featuremanagement.repository;

import com.ecommerce.featuremanagement.domain.FeatureFlag;
import com.ecommerce.featuremanagement.domain.FlagStatus;
import com.ecommerce.featuremanagement.domain.FlagType;
import com.ecommerce.featuremanagement.domain.Variant;
import com.ecommerce.featuremanagement.domain.exception.NotFoundException;
import com.ecommerce.featuremanagement.util.JsonUtils;
import org.springframework.dao.EmptyResultDataAccessException;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.jdbc.core.RowMapper;
import org.springframework.stereotype.Repository;
import org.springframework.transaction.annotation.Transactional;

import java.sql.Array;
import java.sql.Timestamp;
import java.time.Instant;
import java.util.ArrayList;
import java.util.List;
import java.util.UUID;

@Repository
public class FlagRepository {

    private final JdbcTemplate jdbc;
    private final RuleRepository ruleRepository;

    public FlagRepository(JdbcTemplate jdbc, RuleRepository ruleRepository) {
        this.jdbc = jdbc;
        this.ruleRepository = ruleRepository;
    }

    private static final RowMapper<FeatureFlag> MAPPER = (rs, rowNum) -> {
        FeatureFlag f = new FeatureFlag();
        f.id = UUID.fromString(rs.getString("id"));
        f.applicationId = UUID.fromString(rs.getString("application_id"));
        f.environmentId = UUID.fromString(rs.getString("environment_id"));
        f.key = rs.getString("key");
        f.name = rs.getString("name");
        f.description = rs.getString("description");
        f.type = FlagType.fromValue(rs.getString("type"));
        f.status = FlagStatus.fromValue(rs.getString("status"));
        f.defaultValue = JsonUtils.parse(rs.getString("default_value"));
        Array tagsArr = rs.getArray("tags");
        List<String> tags = new ArrayList<>();
        if (tagsArr != null) {
            for (Object o : (Object[]) tagsArr.getArray()) {
                tags.add((String) o);
            }
        }
        f.tags = tags;
        f.version = rs.getLong("version");
        f.createdAt = rs.getTimestamp("created_at").toInstant();
        f.updatedAt = rs.getTimestamp("updated_at").toInstant();
        return f;
    };

    private static final RowMapper<Variant> VARIANT_MAPPER = (rs, rowNum) -> {
        Variant v = new Variant();
        v.id = UUID.fromString(rs.getString("id"));
        v.flagId = UUID.fromString(rs.getString("flag_id"));
        v.key = rs.getString("key");
        v.value = JsonUtils.parse(rs.getString("value"));
        v.description = rs.getString("description");
        return v;
    };

    private void loadRelations(FeatureFlag flag) {
        flag.variants = jdbc.query("SELECT * FROM variants WHERE flag_id = ?", VARIANT_MAPPER, flag.id);
        flag.rules = ruleRepository.listByFlag(flag.id);
    }

    @Transactional
    public FeatureFlag create(FeatureFlag flag) {
        UUID id = UUID.randomUUID();
        Instant now = Instant.now();
        jdbc.update("INSERT INTO feature_flags (id, application_id, environment_id, key, name, description, type, " +
                        "status, default_value, tags, version, created_at, updated_at) " +
                        "VALUES (?,?,?,?,?,?,?,?,?::jsonb,?,1,?,?)",
                id, flag.applicationId, flag.environmentId, flag.key, flag.name, flag.description,
                flag.type.value(), flag.status.value(), JsonUtils.toJson(flag.defaultValue),
                flag.tags.toArray(new String[0]), Timestamp.from(now), Timestamp.from(now));
        flag.id = id;
        flag.version = 1;
        flag.createdAt = now;
        flag.updatedAt = now;
        insertVariants(flag);
        return flag;
    }

    private void insertVariants(FeatureFlag flag) {
        for (Variant v : flag.variants) {
            UUID vid = UUID.randomUUID();
            jdbc.update("INSERT INTO variants (id, flag_id, key, value, description) VALUES (?,?,?,?::jsonb,?)",
                    vid, flag.id, v.key, JsonUtils.toJson(v.value), v.description);
            v.id = vid;
            v.flagId = flag.id;
        }
    }

    public FeatureFlag getByKey(UUID appId, UUID envId, String key) {
        FeatureFlag flag;
        try {
            flag = jdbc.queryForObject(
                    "SELECT * FROM feature_flags WHERE application_id = ? AND environment_id = ? AND key = ? AND deleted_at IS NULL",
                    MAPPER, appId, envId, key);
        } catch (EmptyResultDataAccessException e) {
            throw new NotFoundException("flag not found");
        }
        loadRelations(flag);
        return flag;
    }

    public FeatureFlag getById(UUID id) {
        FeatureFlag flag;
        try {
            flag = jdbc.queryForObject("SELECT * FROM feature_flags WHERE id = ? AND deleted_at IS NULL", MAPPER, id);
        } catch (EmptyResultDataAccessException e) {
            throw new NotFoundException("flag not found");
        }
        loadRelations(flag);
        return flag;
    }

    public PagedResult<FeatureFlag> list(UUID appId, UUID envId, FlagStatus status, List<String> tags, int limit, int offset) {
        StringBuilder sql = new StringBuilder("SELECT * FROM feature_flags WHERE application_id = ? AND environment_id = ? AND deleted_at IS NULL");
        StringBuilder countSql = new StringBuilder("SELECT COUNT(*) FROM feature_flags WHERE application_id = ? AND environment_id = ? AND deleted_at IS NULL");
        List<Object> args = new ArrayList<>(List.of(appId, envId));
        if (status != null) {
            sql.append(" AND status = ?");
            countSql.append(" AND status = ?");
            args.add(status.value());
        }
        if (tags != null && !tags.isEmpty()) {
            sql.append(" AND tags && ?");
            countSql.append(" AND tags && ?");
            args.add(tags.toArray(new String[0]));
        }
        int total = jdbc.queryForObject(countSql.toString(), Integer.class, args.toArray());
        sql.append(" ORDER BY created_at DESC LIMIT ? OFFSET ?");
        args.add(limit);
        args.add(offset);
        List<FeatureFlag> flags = jdbc.query(sql.toString(), MAPPER, args.toArray());
        return new PagedResult<>(flags, total);
    }

    public List<FeatureFlag> listByEnvironment(UUID appId, UUID envId) {
        List<FeatureFlag> flags = jdbc.query(
                "SELECT * FROM feature_flags WHERE application_id = ? AND environment_id = ? AND status = 'active' AND deleted_at IS NULL",
                MAPPER, appId, envId);
        for (FeatureFlag f : flags) {
            loadRelations(f);
        }
        return flags;
    }

    @Transactional
    public FeatureFlag update(FeatureFlag flag) {
        Instant now = Instant.now();
        int rows = jdbc.update(
                "UPDATE feature_flags SET name = ?, description = ?, default_value = ?::jsonb, tags = ?, version = version + 1, updated_at = ? " +
                        "WHERE id = ? AND deleted_at IS NULL",
                flag.name, flag.description, JsonUtils.toJson(flag.defaultValue), flag.tags.toArray(new String[0]),
                Timestamp.from(now), flag.id);
        if (rows == 0) {
            throw new NotFoundException("flag not found");
        }
        flag.version = flag.version + 1;
        flag.updatedAt = now;
        return flag;
    }

    public void delete(UUID id) {
        int rows = jdbc.update("UPDATE feature_flags SET deleted_at = ? WHERE id = ? AND deleted_at IS NULL",
                Timestamp.from(Instant.now()), id);
        if (rows == 0) {
            throw new NotFoundException("flag not found");
        }
    }

    public void updateStatus(UUID id, FlagStatus status) {
        int rows = jdbc.update(
                "UPDATE feature_flags SET status = ?, version = version + 1, updated_at = ? WHERE id = ? AND deleted_at IS NULL",
                status.value(), Timestamp.from(Instant.now()), id);
        if (rows == 0) {
            throw new NotFoundException("flag not found");
        }
    }
}
