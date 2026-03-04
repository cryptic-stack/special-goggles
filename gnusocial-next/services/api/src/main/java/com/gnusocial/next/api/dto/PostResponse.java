package com.gnusocial.next.api.dto;

import java.time.Instant;
import java.util.UUID;

public record PostResponse(
    UUID id,
    UUID authorId,
    String content,
    String visibility,
    UUID replyTo,
    String federationId,
    Instant createdAt) {}

