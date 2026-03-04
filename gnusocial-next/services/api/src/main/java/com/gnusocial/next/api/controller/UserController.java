package com.gnusocial.next.api.controller;

import com.gnusocial.next.api.dto.FollowRequest;
import com.gnusocial.next.api.dto.UserRegistrationRequest;
import com.gnusocial.next.api.dto.UserResponse;
import com.gnusocial.next.api.service.UserService;
import jakarta.validation.Valid;
import java.util.Map;
import java.util.UUID;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.DeleteMapping;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.RestController;

@RestController
@RequestMapping("/api/v1/users")
public class UserController {
  private final UserService userService;

  public UserController(UserService userService) {
    this.userService = userService;
  }

  @PostMapping
  public UserResponse register(@Valid @RequestBody UserRegistrationRequest request) {
    return userService.registerLocal(request);
  }

  @GetMapping("/{username}")
  public UserResponse get(@PathVariable String username) {
    return userService.getByUsername(username).map(userService::toResponse).orElseThrow(() -> new com.gnusocial.next.api.service.NotFoundException("User not found"));
  }

  @PostMapping("/follow")
  public ResponseEntity<Map<String, String>> follow(@Valid @RequestBody FollowRequest request) {
    userService.follow(request.followerId(), request.followedId());
    return ResponseEntity.ok(Map.of("status", "ok"));
  }

  @DeleteMapping("/follow")
  public ResponseEntity<Map<String, String>> unfollow(
      @RequestParam("followerId") UUID followerId, @RequestParam("followedId") UUID followedId) {
    userService.unfollow(followerId, followedId);
    return ResponseEntity.ok(Map.of("status", "ok"));
  }

  @PostMapping("/mute")
  public ResponseEntity<Map<String, String>> mute(@Valid @RequestBody FollowRequest request) {
    userService.mute(request.followerId(), request.followedId());
    return ResponseEntity.ok(Map.of("status", "ok"));
  }

  @DeleteMapping("/mute")
  public ResponseEntity<Map<String, String>> unmute(
      @RequestParam("muterId") UUID muterId, @RequestParam("mutedId") UUID mutedId) {
    userService.unmute(muterId, mutedId);
    return ResponseEntity.ok(Map.of("status", "ok"));
  }

  @PostMapping("/hide-post")
  public ResponseEntity<Map<String, String>> hidePost(
      @RequestParam("userId") UUID userId, @RequestParam("postId") UUID postId) {
    userService.hidePost(userId, postId);
    return ResponseEntity.ok(Map.of("status", "ok"));
  }
}

