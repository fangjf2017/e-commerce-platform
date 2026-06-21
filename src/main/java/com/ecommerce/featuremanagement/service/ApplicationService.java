package com.ecommerce.featuremanagement.service;

import com.ecommerce.featuremanagement.domain.Application;
import com.ecommerce.featuremanagement.domain.Environment;
import com.ecommerce.featuremanagement.repository.ApplicationRepository;
import com.ecommerce.featuremanagement.repository.EnvironmentRepository;
import com.ecommerce.featuremanagement.repository.PagedResult;
import com.ecommerce.featuremanagement.service.dto.CreateApplicationInput;
import com.ecommerce.featuremanagement.service.dto.CreateEnvironmentInput;
import com.ecommerce.featuremanagement.service.dto.UpdateApplicationInput;
import com.ecommerce.featuremanagement.service.dto.UpdateEnvironmentInput;
import org.springframework.stereotype.Service;

import java.util.UUID;

@Service
public class ApplicationService {

    private final ApplicationRepository apps;
    private final EnvironmentRepository envs;

    public ApplicationService(ApplicationRepository apps, EnvironmentRepository envs) {
        this.apps = apps;
        this.envs = envs;
    }

    public Application create(CreateApplicationInput inp) {
        Application app = new Application();
        app.name = inp.name;
        app.slug = inp.slug;
        app.description = inp.description != null ? inp.description : "";
        return apps.create(app);
    }

    public Application get(UUID id) {
        return apps.getById(id);
    }

    public Application getBySlug(String slug) {
        return apps.getBySlug(slug);
    }

    public PagedResult<Application> list(int limit, int offset) {
        if (limit <= 0) {
            limit = 20;
        }
        return apps.list(limit, offset);
    }

    public Application update(UUID id, UpdateApplicationInput inp) {
        Application app = apps.getById(id);
        app.name = inp.name;
        app.description = inp.description != null ? inp.description : "";
        return apps.update(app);
    }

    public void delete(UUID id) {
        apps.delete(id);
    }

    public Environment createEnvironment(CreateEnvironmentInput inp) {
        Environment env = new Environment();
        env.applicationId = inp.applicationId;
        env.name = inp.name;
        env.slug = inp.slug;
        env.requiresApproval = inp.requiresApproval;
        return envs.create(env);
    }

    public Environment getEnvironment(UUID envId) {
        return envs.getById(envId);
    }

    public PagedResult<Environment> listEnvironments(UUID appId, int limit, int offset) {
        var all = envs.list(appId);
        int total = all.size();
        if (limit <= 0) {
            limit = 20;
        }
        if (offset < 0) {
            offset = 0;
        }
        if (offset >= total) {
            return new PagedResult<>(java.util.List.of(), total);
        }
        int end = Math.min(offset + limit, total);
        return new PagedResult<>(all.subList(offset, end), total);
    }

    public Environment updateEnvironment(UUID envId, UpdateEnvironmentInput inp) {
        Environment env = envs.getById(envId);
        env.name = inp.name;
        env.requiresApproval = inp.requiresApproval;
        return envs.update(env);
    }

    public void deleteEnvironment(UUID envId) {
        envs.delete(envId);
    }
}
