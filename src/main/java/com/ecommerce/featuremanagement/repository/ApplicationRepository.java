package com.ecommerce.featuremanagement.repository;

import com.ecommerce.featuremanagement.domain.Application;
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
public class ApplicationRepository {

    private final JdbcTemplate jdbc;

    public ApplicationRepository(JdbcTemplate jdbc) {
        this.jdbc = jdbc;
    }

    private static final RowMapper<Application> MAPPER = (rs, rowNum) -> {
        Application a = new Application();
        a.id = UUID.fromString(rs.getString("id"));
        a.name = rs.getString("name");
        a.slug = rs.getString("slug");
        a.description = rs.getString("description");
        a.createdAt = rs.getTimestamp("created_at").toInstant();
        a.updatedAt = rs.getTimestamp("updated_at").toInstant();
        return a;
    };

    public Application create(Application app) {
        UUID id = UUID.randomUUID();
        Instant now = Instant.now();
        jdbc.update(
                "INSERT INTO applications (id, name, slug, description, created_at, updated_at) VALUES (?,?,?,?,?,?)",
                id, app.name, app.slug, app.description, Timestamp.from(now), Timestamp.from(now));
        app.id = id;
        app.createdAt = now;
        app.updatedAt = now;
        return app;
    }

    public Application getById(UUID id) {
        try {
            return jdbc.queryForObject(
                    "SELECT * FROM applications WHERE id = ? AND deleted_at IS NULL", MAPPER, id);
        } catch (EmptyResultDataAccessException e) {
            throw new NotFoundException("application not found");
        }
    }

    public Application getBySlug(String slug) {
        try {
            return jdbc.queryForObject(
                    "SELECT * FROM applications WHERE slug = ? AND deleted_at IS NULL", MAPPER, slug);
        } catch (EmptyResultDataAccessException e) {
            throw new NotFoundException("application not found");
        }
    }

    public PagedResult<Application> list(int limit, int offset) {
        List<Application> apps = jdbc.query(
                "SELECT * FROM applications WHERE deleted_at IS NULL ORDER BY created_at DESC LIMIT ? OFFSET ?",
                MAPPER, limit, offset);
        int total = jdbc.queryForObject(
                "SELECT COUNT(*) FROM applications WHERE deleted_at IS NULL", Integer.class);
        return new PagedResult<>(apps, total);
    }

    public Application update(Application app) {
        Instant now = Instant.now();
        int rows = jdbc.update(
                "UPDATE applications SET name = ?, description = ?, updated_at = ? WHERE id = ? AND deleted_at IS NULL",
                app.name, app.description, Timestamp.from(now), app.id);
        if (rows == 0) {
            throw new NotFoundException("application not found");
        }
        app.updatedAt = now;
        return app;
    }

    public void delete(UUID id) {
        int rows = jdbc.update("UPDATE applications SET deleted_at = ? WHERE id = ? AND deleted_at IS NULL",
                Timestamp.from(Instant.now()), id);
        if (rows == 0) {
            throw new NotFoundException("application not found");
        }
    }
}
