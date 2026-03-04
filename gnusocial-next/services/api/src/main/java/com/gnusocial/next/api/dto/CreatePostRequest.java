package com.gnusocial.next.api.dto;

import jakarta.validation.constraints.NotBlank;
import jakarta.validation.constraints.NotNull;
import jakarta.validation.constraints.Size;
import java.util.UUID;

public record CreatePostRequest(
    @NotNull UUID authorId,
    @NotBlank @Size(max = 2000) String content,
    @NotBlank String visibility,
    UUID replyTo) {}

