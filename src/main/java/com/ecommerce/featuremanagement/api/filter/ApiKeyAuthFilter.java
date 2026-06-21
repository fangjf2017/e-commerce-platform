package com.ecommerce.featuremanagement.api.filter;

import com.ecommerce.featuremanagement.domain.AuditActor;
import jakarta.servlet.FilterChain;
import jakarta.servlet.ServletException;
import jakarta.servlet.http.HttpServletRequest;
import jakarta.servlet.http.HttpServletResponse;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Component;
import org.springframework.web.filter.OncePerRequestFilter;

import java.io.IOException;

/** Validates the X-API-Key header on /api/** routes, mirroring the Go auth middleware. */
@Component
public class ApiKeyAuthFilter extends OncePerRequestFilter {

    public static final String ACTOR_ATTRIBUTE = "actor";

    private final String apiKey;

    public ApiKeyAuthFilter(@Value("${fms.auth.api-key}") String apiKey) {
        this.apiKey = apiKey;
    }

    @Override
    protected boolean shouldNotFilter(HttpServletRequest request) {
        return !request.getRequestURI().startsWith("/api/");
    }

    @Override
    protected void doFilterInternal(HttpServletRequest request, HttpServletResponse response, FilterChain chain)
            throws ServletException, IOException {
        String provided = request.getHeader("X-API-Key");
        if (provided == null || !provided.equals(apiKey)) {
            response.setStatus(HttpServletResponse.SC_UNAUTHORIZED);
            response.setContentType("application/json");
            response.getWriter().write("{\"error\":{\"code\":\"unauthorized\",\"message\":\"missing or invalid API key\"}}");
            return;
        }
        request.setAttribute(ACTOR_ATTRIBUTE, new AuditActor(provided, "", "api_key"));
        chain.doFilter(request, response);
    }
}
