package com.ecommerce.featuremanagement.repository;

import com.ecommerce.featuremanagement.domain.AuditAction;
import com.ecommerce.featuremanagement.domain.AuditActor;
import com.ecommerce.featuremanagement.domain.AuditEvent;
import com.ecommerce.featuremanagement.domain.exception.NotFoundException;
import com.ecommerce.featuremanagement.util.JsonUtils;
import org.springframework.dao.EmptyResultDataAccessException;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.jdbc.core.RowMapper;
import org.springframework.stereotype.Repository;

import java.sql.Timestamp;
import java.time.Instant;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.UUID;

@Repository
public class AuditRepository {

    private final JdbcTemplate jdbc;

    public AuditRepository(JdbcTemplate jdbc) {
        this.jdbc = jdbc;
    }

    private static final RowMapper<AuditEvent> MAPPER = (rs, rowNum) -> {
        AuditEvent e = new AuditEvent();
        e.id = UUID.fromString(rs.getString("id"));
        e.applicationId = UUID.fromString(rs.getString("application_id"));
        String envId = rs.getString("environment_id");
        e.environmentId = envId == null ? null : UUID.fromString(envId);
        e.action = AuditAction.fromValue(rs.getString("action"));
        e.resourceType = rs.getString("resource_type");
        e.resourceId = UUID.fromString(rs.getString("resource_id"));
        e.resourceKey = rs.getString("resource_key");
        e.actor = new AuditActor(rs.getString("actor_id"), rs.getString("actor_email"), rs.getString("actor_type"));
        String before = rs.getString("before_state");
        e.before = before == null ? null : JsonUtils.parse(before);
        String after = rs.getString("after_state");
        e.after = after == null ? null : JsonUtils.parse(after);
        String metadataJson = rs.getString("metadata");
        Map<String, String> metadata = new HashMap<>();
        if (metadataJson != null) {
            JsonUtils.parse(metadataJson).fields().forEachRemaining(
                    entry -> metadata.put(entry.getKey(), entry.getValue().asText()));
        }
        e.metadata = metadata;
        e.occurredAt = rs.getTimestamp("occurred_at").toInstant();
        return e;
    };

    public AuditEvent create(AuditEvent event) {
        if (event.id == null) {
            event.id = UUID.randomUUID();
        }
        if (event.occurredAt == null) {
            event.occurredAt = Instant.now();
        }
        jdbc.update("INSERT INTO audit_events (id, application_id, environment_id, action, resource_type, resource_id, " +
                        "resource_key, actor_id, actor_email, actor_type, before_state, after_state, metadata, occurred_at) " +
                        "VALUES (?,?,?,?,?,?,?,?,?,?,?::jsonb,?::jsonb,?::jsonb,?)",
                event.id, event.applicationId, event.environmentId, event.action.value(), event.resourceType,
                event.resourceId, event.resourceKey, event.actor.id, event.actor.email, event.actor.type,
                event.before == null ? null : JsonUtils.toJson(event.before),
                event.after == null ? null : JsonUtils.toJson(event.after),
                JsonUtils.toJson(event.metadata == null ? new HashMap<>() : event.metadata),
                Timestamp.from(event.occurredAt));
        return event;
    }

    public PagedResult<AuditEvent> list(UUID appId, ListAuditFilter filter) {
        StringBuilder sql = new StringBuilder("SELECT * FROM audit_events WHERE application_id = ?");
        StringBuilder countSql = new StringBuilder("SELECT COUNT(*) FROM audit_events WHERE application_id = ?");
        List<Object> args = new ArrayList<>();
        args.add(appId);
        if (filter.resourceType != null && !filter.resourceType.isBlank()) {
            sql.append(" AND resource_type = ?");
            countSql.append(" AND resource_type = ?");
            args.add(filter.resourceType);
        }
        if (filter.action != null && !filter.action.isBlank()) {
            sql.append(" AND action = ?");
            countSql.append(" AND action = ?");
            args.add(filter.action);
        }
        if (filter.actorId != null && !filter.actorId.isBlank()) {
            sql.append(" AND actor_id = ?");
            countSql.append(" AND actor_id = ?");
            args.add(filter.actorId);
        }
        if (filter.after != null) {
            sql.append(" AND occurred_at > ?");
            countSql.append(" AND occurred_at > ?");
            args.add(Timestamp.from(filter.after));
        }
        if (filter.before != null) {
            sql.append(" AND occurred_at < ?");
            countSql.append(" AND occurred_at < ?");
            args.add(Timestamp.from(filter.before));
        }
        int total = jdbc.queryForObject(countSql.toString(), Integer.class, args.toArray());
        int limit = filter.limit > 0 ? filter.limit : 50;
        sql.append(" ORDER BY occurred_at DESC LIMIT ? OFFSET ?");
        args.add(limit);
        args.add(filter.offset);
        List<AuditEvent> events = jdbc.query(sql.toString(), MAPPER, args.toArray());
        return new PagedResult<>(events, total);
    }

    public AuditEvent getById(UUID id) {
        try {
            return jdbc.queryForObject("SELECT * FROM audit_events WHERE id = ?", MAPPER, id);
        } catch (EmptyResultDataAccessException e) {
            throw new NotFoundException("audit event not found");
        }
    }
}
