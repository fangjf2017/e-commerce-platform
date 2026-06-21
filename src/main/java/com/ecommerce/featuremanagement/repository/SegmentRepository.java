package com.ecommerce.featuremanagement.repository;

import com.ecommerce.featuremanagement.domain.ConditionOperator;
import com.ecommerce.featuremanagement.domain.Segment;
import com.ecommerce.featuremanagement.domain.SegmentOperator;
import com.ecommerce.featuremanagement.domain.SegmentRule;
import com.ecommerce.featuremanagement.domain.exception.NotFoundException;
import com.ecommerce.featuremanagement.util.JsonUtils;
import org.springframework.dao.EmptyResultDataAccessException;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.jdbc.core.RowMapper;
import org.springframework.stereotype.Repository;
import org.springframework.transaction.annotation.Transactional;

import java.sql.Timestamp;
import java.time.Instant;
import java.util.List;
import java.util.UUID;

@Repository
public class SegmentRepository {

    private final JdbcTemplate jdbc;

    public SegmentRepository(JdbcTemplate jdbc) {
        this.jdbc = jdbc;
    }

    private static final RowMapper<Segment> MAPPER = (rs, rowNum) -> {
        Segment s = new Segment();
        s.id = UUID.fromString(rs.getString("id"));
        s.applicationId = UUID.fromString(rs.getString("application_id"));
        s.name = rs.getString("name");
        s.description = rs.getString("description");
        s.operator = SegmentOperator.fromValue(rs.getString("operator"));
        s.version = rs.getLong("version");
        s.createdAt = rs.getTimestamp("created_at").toInstant();
        s.updatedAt = rs.getTimestamp("updated_at").toInstant();
        return s;
    };

    private static final RowMapper<SegmentRule> RULE_MAPPER = (rs, rowNum) -> {
        SegmentRule r = new SegmentRule();
        r.id = UUID.fromString(rs.getString("id"));
        r.segmentId = UUID.fromString(rs.getString("segment_id"));
        r.attribute = rs.getString("attribute");
        r.operator = ConditionOperator.fromValue(rs.getString("operator"));
        r.value = JsonUtils.parse(rs.getString("value"));
        return r;
    };

    @Transactional
    public Segment create(Segment seg) {
        UUID id = UUID.randomUUID();
        Instant now = Instant.now();
        if (seg.operator == null) {
            seg.operator = SegmentOperator.ALL;
        }
        jdbc.update(
                "INSERT INTO segments (id, application_id, name, description, operator, version, created_at, updated_at) VALUES (?,?,?,?,?,1,?,?)",
                id, seg.applicationId, seg.name, seg.description, seg.operator.value(), Timestamp.from(now), Timestamp.from(now));
        seg.id = id;
        seg.version = 1;
        seg.createdAt = now;
        seg.updatedAt = now;
        insertRules(seg);
        return seg;
    }

    private void insertRules(Segment seg) {
        for (SegmentRule r : seg.rules) {
            UUID rid = UUID.randomUUID();
            jdbc.update("INSERT INTO segment_rules (id, segment_id, attribute, operator, value) VALUES (?,?,?,?,?::jsonb)",
                    rid, seg.id, r.attribute, r.operator.value(), JsonUtils.toJson(r.value));
            r.id = rid;
            r.segmentId = seg.id;
        }
    }

    public Segment getById(UUID segId) {
        Segment seg;
        try {
            seg = jdbc.queryForObject("SELECT * FROM segments WHERE id = ?", MAPPER, segId);
        } catch (EmptyResultDataAccessException e) {
            throw new NotFoundException("segment not found");
        }
        seg.rules = jdbc.query("SELECT * FROM segment_rules WHERE segment_id = ? ORDER BY created_at", RULE_MAPPER, segId);
        return seg;
    }

    public List<Segment> list(UUID appId) {
        List<Segment> segs = jdbc.query(
                "SELECT * FROM segments WHERE application_id = ? ORDER BY created_at DESC", MAPPER, appId);
        for (Segment s : segs) {
            s.rules = jdbc.query("SELECT * FROM segment_rules WHERE segment_id = ? ORDER BY created_at", RULE_MAPPER, s.id);
        }
        return segs;
    }

    @Transactional
    public Segment update(Segment seg) {
        Instant now = Instant.now();
        int rows = jdbc.update(
                "UPDATE segments SET name = ?, description = ?, operator = ?, version = version + 1, updated_at = ? WHERE id = ?",
                seg.name, seg.description, seg.operator.value(), Timestamp.from(now), seg.id);
        if (rows == 0) {
            throw new NotFoundException("segment not found");
        }
        jdbc.update("DELETE FROM segment_rules WHERE segment_id = ?", seg.id);
        insertRules(seg);
        seg.version = seg.version + 1;
        seg.updatedAt = now;
        return seg;
    }

    public void delete(UUID id) {
        int rows = jdbc.update("DELETE FROM segments WHERE id = ?", id);
        if (rows == 0) {
            throw new NotFoundException("segment not found");
        }
    }
}
