package com.ecommerce.featuremanagement.repository;

import com.ecommerce.featuremanagement.domain.Environment;
import com.ecommerce.featuremanagement.domain.exception.NotFoundException;
import org.springframework.dao.EmptyResultDataAccessException;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.jdbc.core.RowMapper;
import org.springframework.stereotype.Repository;

import java.sql.Timestamp;
import java.time.Instant;
import java.util.List;
import java.util.UUID;

@Repository
public class EnvironmentRepository {

    private final JdbcTemplate jdbc;

    public EnvironmentRepository(JdbcTemplate jdbc) {
        this.jdbc = jdbc;
    }

    private static final RowMapper<Environment> MAPPER = (rs, rowNum) -> {
        Environment e = new Environment();
        e.id = UUID.fromString(rs.getString("id"));
        e.applicationId = UUID.fromString(rs.getString("application_id"));
        e.name = rs.getString("name");
        e.slug = rs.getString("slug");
        e.requiresApproval = rs.getBoolean("requires_approval");
        e.createdAt = rs.getTimestamp("created_at").toInstant();
        e.updatedAt = rs.getTimestamp("updated_at").toInstant();
        return e;
    };

    public Environment create(Environment env) {
        UUID id = UUID.randomUUID();
        Instant now = Instant.now();
        jdbc.update(
                "INSERT INTO environments (id, application_id, name, slug, requires_approval, created_at, updated_at) VALUES (?,?,?,?,?,?,?)",
                id, env.applicationId, env.name, env.slug, env.requiresApproval, Timestamp.from(now), Timestamp.from(now));
        env.id = id;
        env.createdAt = now;
        env.updatedAt = now;
        return env;
    }

    /** Returns an environment by its ID only, regardless of owning application. */
    public Environment getById(UUID envId) {
        try {
            return jdbc.queryForObject(
                    "SELECT * FROM environments WHERE id = ? AND deleted_at IS NULL", MAPPER, envId);
        } catch (EmptyResultDataAccessException e) {
            throw new NotFoundException("environment not found");
        }
    }

    /** Returns an environment scoped to a specific application. */
    public Environment getById(UUID appId, UUID envId) {
        try {
            return jdbc.queryForObject(
                    "SELECT * FROM environments WHERE id = ? AND application_id = ? AND deleted_at IS NULL",
                    MAPPER, envId, appId);
        } catch (EmptyResultDataAccessException e) {
            throw new NotFoundException("environment not found");
        }
    }

    public List<Environment> list(UUID appId) {
        return jdbc.query(
                "SELECT * FROM environments WHERE application_id = ? AND deleted_at IS NULL ORDER BY created_at",
                MAPPER, appId);
    }

    public Environment update(Environment env) {
        Instant now = Instant.now();
        int rows = jdbc.update(
                "UPDATE environments SET name = ?, requires_approval = ?, updated_at = ? WHERE id = ? AND deleted_at IS NULL",
                env.name, env.requiresApproval, Timestamp.from(now), env.id);
        if (rows == 0) {
            throw new NotFoundException("environment not found");
        }
        env.updatedAt = now;
        return env;
    }

    public void delete(UUID envId) {
        int rows = jdbc.update("UPDATE environments SET deleted_at = ? WHERE id = ? AND deleted_at IS NULL",
                Timestamp.from(Instant.now()), envId);
        if (rows == 0) {
            throw new NotFoundException("environment not found");
        }
    }
}
