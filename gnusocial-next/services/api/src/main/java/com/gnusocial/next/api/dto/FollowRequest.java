package com.gnusocial.next.api.dto;

import jakarta.validation.constraints.NotNull;
import java.util.UUID;

public record FollowRequest(@NotNull UUID followerId, @NotNull UUID followedId) {}

