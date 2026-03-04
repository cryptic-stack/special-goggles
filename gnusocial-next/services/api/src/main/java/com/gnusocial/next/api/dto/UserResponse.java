package com.gnusocial.next.api.dto;

import java.time.Instant;
import java.util.UUID;

public record UserResponse(
    UUID id,
    String username,
    String domain,
    String displayName,
    String bio,
    String avatarUrl,
    Instant createdAt) {}

