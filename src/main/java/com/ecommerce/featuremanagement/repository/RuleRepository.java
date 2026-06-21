package com.ecommerce.featuremanagement.repository;

import com.ecommerce.featuremanagement.domain.Condition;
import com.ecommerce.featuremanagement.domain.ConditionOperator;
import com.ecommerce.featuremanagement.domain.Rule;
import com.ecommerce.featuremanagement.domain.RuleType;
import com.ecommerce.featuremanagement.domain.VariantAllocation;
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
public class RuleRepository {

    private final JdbcTemplate jdbc;

    public RuleRepository(JdbcTemplate jdbc) {
        this.jdbc = jdbc;
    }

    private static final RowMapper<Rule> MAPPER = (rs, rowNum) -> {
        Rule r = new Rule();
        r.id = UUID.fromString(rs.getString("id"));
        r.flagId = UUID.fromString(rs.getString("flag_id"));
        r.type = RuleType.fromValue(rs.getString("type"));
        r.priority = rs.getInt("priority");
        r.name = rs.getString("name");
        r.description = rs.getString("description");
        String segId = rs.getString("segment_id");
        r.segmentId = segId == null ? null : UUID.fromString(segId);
        int rolloutPct = rs.getInt("rollout_pct");
        r.rolloutPct = rs.wasNull() ? null : rolloutPct;
        r.rolloutSalt = UUID.fromString(rs.getString("rollout_salt"));
        Timestamp ss = rs.getTimestamp("schedule_start");
        r.scheduleStart = ss == null ? null : ss.toInstant();
        Timestamp se = rs.getTimestamp("schedule_end");
        r.scheduleEnd = se == null ? null : se.toInstant();
        r.enabled = rs.getBoolean("enabled");
        r.createdAt = rs.getTimestamp("created_at").toInstant();
        r.updatedAt = rs.getTimestamp("updated_at").toInstant();
        return r;
    };

    private static final RowMapper<Condition> CONDITION_MAPPER = (rs, rowNum) -> {
        Condition c = new Condition();
        c.id = UUID.fromString(rs.getString("id"));
        c.ruleId = UUID.fromString(rs.getString("rule_id"));
        c.attribute = rs.getString("attribute");
        c.operator = ConditionOperator.fromValue(rs.getString("operator"));
        c.value = JsonUtils.parse(rs.getString("value"));
        c.negate = rs.getBoolean("negate");
        return c;
    };

    private static final RowMapper<VariantAllocation> ALLOCATION_MAPPER = (rs, rowNum) -> {
        VariantAllocation a = new VariantAllocation();
        a.id = UUID.fromString(rs.getString("id"));
        a.ruleId = UUID.fromString(rs.getString("rule_id"));
        a.variantId = UUID.fromString(rs.getString("variant_id"));
        a.rolloutFrom = rs.getInt("rollout_from");
        a.rolloutTo = rs.getInt("rollout_to");
        return a;
    };

    private void loadRelations(Rule rule) {
        rule.conditions = jdbc.query("SELECT * FROM conditions WHERE rule_id = ? ORDER BY created_at", CONDITION_MAPPER, rule.id);
        rule.variants = jdbc.query("SELECT * FROM variant_allocations WHERE rule_id = ?", ALLOCATION_MAPPER, rule.id);
    }

    @Transactional
    public Rule create(Rule rule) {
        UUID id = UUID.randomUUID();
        Instant now = Instant.now();
        if (rule.rolloutSalt == null) {
            rule.rolloutSalt = UUID.randomUUID();
        }
        jdbc.update("INSERT INTO rules (id, flag_id, type, priority, name, description, segment_id, rollout_pct, " +
                        "rollout_salt, schedule_start, schedule_end, enabled, created_at, updated_at) " +
                        "VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?)",
                id, rule.flagId, rule.type.value(), rule.priority, rule.name, rule.description, rule.segmentId,
                rule.rolloutPct, rule.rolloutSalt,
                rule.scheduleStart == null ? null : Timestamp.from(rule.scheduleStart),
                rule.scheduleEnd == null ? null : Timestamp.from(rule.scheduleEnd),
                rule.enabled, Timestamp.from(now), Timestamp.from(now));
        rule.id = id;
        rule.createdAt = now;
        rule.updatedAt = now;
        insertConditions(rule);
        insertAllocations(rule);
        return rule;
    }

    private void insertConditions(Rule rule) {
        for (Condition c : rule.conditions) {
            UUID cid = UUID.randomUUID();
            jdbc.update("INSERT INTO conditions (id, rule_id, attribute, operator, value, negate) VALUES (?,?,?,?,?::jsonb,?)",
                    cid, rule.id, c.attribute, c.operator.value(), JsonUtils.toJson(c.value), c.negate);
            c.id = cid;
            c.ruleId = rule.id;
        }
    }

    private void insertAllocations(Rule rule) {
        for (VariantAllocation a : rule.variants) {
            UUID aid = UUID.randomUUID();
            jdbc.update("INSERT INTO variant_allocations (id, rule_id, variant_id, rollout_from, rollout_to) VALUES (?,?,?,?,?)",
                    aid, rule.id, a.variantId, a.rolloutFrom, a.rolloutTo);
            a.id = aid;
            a.ruleId = rule.id;
        }
    }

    public Rule getById(UUID ruleId) {
        Rule rule;
        try {
            rule = jdbc.queryForObject("SELECT * FROM rules WHERE id = ?", MAPPER, ruleId);
        } catch (EmptyResultDataAccessException e) {
            throw new NotFoundException("rule not found");
        }
        loadRelations(rule);
        return rule;
    }

    public List<Rule> listByFlag(UUID flagId) {
        List<Rule> rules = jdbc.query("SELECT * FROM rules WHERE flag_id = ? ORDER BY priority", MAPPER, flagId);
        for (Rule r : rules) {
            loadRelations(r);
        }
        return rules;
    }

    @Transactional
    public Rule update(Rule rule) {
        Instant now = Instant.now();
        int rows = jdbc.update("UPDATE rules SET type = ?, priority = ?, name = ?, description = ?, segment_id = ?, " +
                        "rollout_pct = ?, schedule_start = ?, schedule_end = ?, enabled = ?, updated_at = ? WHERE id = ?",
                rule.type.value(), rule.priority, rule.name, rule.description, rule.segmentId, rule.rolloutPct,
                rule.scheduleStart == null ? null : Timestamp.from(rule.scheduleStart),
                rule.scheduleEnd == null ? null : Timestamp.from(rule.scheduleEnd),
                rule.enabled, Timestamp.from(now), rule.id);
        if (rows == 0) {
            throw new NotFoundException("rule not found");
        }
        jdbc.update("DELETE FROM conditions WHERE rule_id = ?", rule.id);
        jdbc.update("DELETE FROM variant_allocations WHERE rule_id = ?", rule.id);
        insertConditions(rule);
        insertAllocations(rule);
        rule.updatedAt = now;
        return rule;
    }

    public void delete(UUID ruleId) {
        int rows = jdbc.update("DELETE FROM rules WHERE id = ?", ruleId);
        if (rows == 0) {
            throw new NotFoundException("rule not found");
        }
    }

    @Transactional
    public void reorderByPriority(List<UUID> orderedRuleIds) {
        for (int i = 0; i < orderedRuleIds.size(); i++) {
            jdbc.update("UPDATE rules SET priority = ?, updated_at = ? WHERE id = ?",
                    i, Timestamp.from(Instant.now()), orderedRuleIds.get(i));
        }
    }
}
