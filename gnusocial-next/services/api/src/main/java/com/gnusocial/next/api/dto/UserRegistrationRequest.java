package com.gnusocial.next.api.dto;

import jakarta.validation.constraints.Email;
import jakarta.validation.constraints.NotBlank;
import jakarta.validation.constraints.Size;

public record UserRegistrationRequest(
    @NotBlank @Size(min = 3, max = 32) String username,
    @Email @NotBlank String email,
    @NotBlank @Size(min = 8, max = 200) String password,
    @NotBlank @Size(min = 1, max = 100) String displayName,
    @Size(max = 500) String bio,
    String avatarUrl) {}

