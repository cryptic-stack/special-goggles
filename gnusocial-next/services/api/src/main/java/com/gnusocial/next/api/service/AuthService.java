package com.gnusocial.next.api.service;

import com.gnusocial.next.api.dto.LoginRequest;
import com.gnusocial.next.api.dto.LoginResponse;
import com.gnusocial.next.api.dto.UserResponse;
import com.gnusocial.next.api.model.UserEntity;
import java.time.Duration;
import java.util.UUID;
import org.springframework.data.redis.core.StringRedisTemplate;
import org.springframework.stereotype.Service;

@Service
public class AuthService {
  private static final Duration SESSION_TTL = Duration.ofDays(14);

  private final UserService userService;
  private final PasswordService passwordService;
  private final StringRedisTemplate redisTemplate;

  public AuthService(
      UserService userService, PasswordService passwordService, StringRedisTemplate redisTemplate) {
    this.userService = userService;
    this.passwordService = passwordService;
    this.redisTemplate = redisTemplate;
  }

  public LoginResponse login(LoginRequest request) {
    UserEntity user =
        userService
            .getByEmail(request.email())
            .orElseThrow(() -> new IllegalArgumentException("Invalid credentials"));

    if (user.getPasswordHash() == null
        || !passwordService.matches(request.password(), user.getPasswordHash())) {
      throw new IllegalArgumentException("Invalid credentials");
    }

    String sessionToken = UUID.randomUUID().toString();
    redisTemplate.opsForValue().set("session:" + sessionToken, user.getId().toString(), SESSION_TTL);

    UserResponse response = userService.toResponse(user);
    return new LoginResponse(sessionToken, response);
  }

  public void logout(String sessionToken) {
    if (sessionToken == null || sessionToken.isBlank()) {
      return;
    }
    redisTemplate.delete("session:" + sessionToken);
  }
}

