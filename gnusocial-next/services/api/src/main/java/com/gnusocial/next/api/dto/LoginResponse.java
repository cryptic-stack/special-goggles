package com.gnusocial.next.api.dto;

public record LoginResponse(String sessionToken, UserResponse user) {}

