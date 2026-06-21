package com.ecommerce.featuremanagement.api.controller;

import com.ecommerce.featuremanagement.api.dto.ApiResponse;
import com.ecommerce.featuremanagement.domain.AuditActor;
import com.ecommerce.featuremanagement.domain.Rule;
import com.ecommerce.featuremanagement.service.FlagService;
import com.ecommerce.featuremanagement.service.dto.CreateRuleInput;
import com.ecommerce.featuremanagement.service.dto.UpdateRuleInput;
import jakarta.servlet.http.HttpServletRequest;
import org.springframework.web.bind.annotation.*;

import java.util.List;
import java.util.Map;
import java.util.UUID;

@RestController
@RequestMapping("/api/v1")
public class RuleController {

    private final FlagService flags;

    public RuleController(FlagService flags) {
        this.flags = flags;
    }

    @PostMapping("/flags/{flagId}/rules")
    public ApiResponse<Rule> create(@PathVariable UUID flagId, @RequestBody CreateRuleInput inp, HttpServletRequest req) {
        inp.flagId = flagId;
        inp.actor = actor(req);
        return ApiResponse.of(flags.createRule(inp));
    }

    @PutMapping("/rules/{ruleId}")
    public ApiResponse<Rule> update(@PathVariable UUID ruleId, @RequestBody UpdateRuleInput inp, HttpServletRequest req) {
        inp.actor = actor(req);
        return ApiResponse.of(flags.updateRule(ruleId, inp));
    }

    @DeleteMapping("/rules/{ruleId}")
    public void delete(@PathVariable UUID ruleId, HttpServletRequest req) {
        flags.deleteRule(ruleId, actor(req));
    }

    @PostMapping("/flags/{flagId}/rules/reorder")
    public void reorder(@PathVariable UUID flagId, @RequestBody Map<String, List<UUID>> body) {
        flags.reorderRules(flagId, body.get("ruleIds"));
    }

    private AuditActor actor(HttpServletRequest req) {
        Object attr = req.getAttribute("actor");
        return attr instanceof AuditActor ? (AuditActor) attr : new AuditActor("unknown", null, "system");
    }
}
