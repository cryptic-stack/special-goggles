package com.gnusocial.next.api.service;

import com.gnusocial.next.api.dto.UserRegistrationRequest;
import com.gnusocial.next.api.dto.UserResponse;
import com.gnusocial.next.api.model.FollowEntity;
import com.gnusocial.next.api.model.UserEntity;
import com.gnusocial.next.api.repository.FollowRepository;
import com.gnusocial.next.api.repository.UserRepository;
import java.net.URI;
import java.time.Clock;
import java.time.Instant;
import java.util.List;
import java.util.Optional;
import java.util.UUID;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

@Service
public class UserService {
  private final UserRepository userRepository;
  private final FollowRepository followRepository;
  private final PasswordService passwordService;
  private final KeyPairService keyPairService;
  private final JdbcTemplate jdbcTemplate;
  private final Clock clock;

  public UserService(
      UserRepository userRepository,
      FollowRepository followRepository,
      PasswordService passwordService,
      KeyPairService keyPairService,
      JdbcTemplate jdbcTemplate,
      Clock clock) {
    this.userRepository = userRepository;
    this.followRepository = followRepository;
    this.passwordService = passwordService;
    this.keyPairService = keyPairService;
    this.jdbcTemplate = jdbcTemplate;
    this.clock = clock;
  }

  @Transactional
  public UserResponse registerLocal(UserRegistrationRequest request) {
    userRepository.findByUsernameAndDomainIsNull(request.username()).ifPresent(user -> {
      throw new IllegalArgumentException("Username already taken");
    });
    userRepository.findByEmail(request.email()).ifPresent(user -> {
      throw new IllegalArgumentException("Email already registered");
    });

    Instant now = Instant.now(clock);
    KeyPairService.GeneratedKeys keys = keyPairService.generate();

    UserEntity entity = new UserEntity();
    entity.setId(UUID.randomUUID());
    entity.setUsername(request.username());
    entity.setDomain(null);
    entity.setEmail(request.email());
    entity.setPasswordHash(passwordService.hash(request.password()));
    entity.setDisplayName(request.displayName());
    entity.setBio(request.bio() == null ? "" : request.bio());
    entity.setAvatarUrl(request.avatarUrl() == null ? "" : request.avatarUrl());
    entity.setPublicKey(keys.publicKey());
    entity.setPrivateKey(keys.privateKey());
    entity.setCreatedAt(now);

    return toResponse(userRepository.save(entity));
  }

  public UserEntity getRequiredById(UUID userId) {
    return userRepository.findById(userId).orElseThrow(() -> new NotFoundException("User not found"));
  }

  public UserEntity getRequiredLocalByUsername(String username) {
    return userRepository.findByUsernameAndDomainIsNull(username).orElseThrow(() -> new NotFoundException("User not found"));
  }

  public Optional<UserEntity> getByEmail(String email) {
    return userRepository.findByEmail(email);
  }

  public Optional<UserEntity> getByUsername(String username) {
    return userRepository.findByUsernameAndDomainIsNull(username);
  }

  public UserResponse toResponse(UserEntity user) {
    return new UserResponse(
        user.getId(),
        user.getUsername(),
        user.getDomain(),
        user.getDisplayName(),
        user.getBio(),
        user.getAvatarUrl(),
        user.getCreatedAt());
  }

  @Transactional
  public void follow(UUID followerId, UUID followedId) {
    if (followerId.equals(followedId)) {
      throw new IllegalArgumentException("Cannot follow self");
    }
    getRequiredById(followerId);
    getRequiredById(followedId);

    if (followRepository.findByFollowerIdAndFollowedId(followerId, followedId).isPresent()) {
      return;
    }

    FollowEntity follow = new FollowEntity();
    follow.setId(UUID.randomUUID());
    follow.setFollowerId(followerId);
    follow.setFollowedId(followedId);
    follow.setCreatedAt(Instant.now(clock));
    followRepository.save(follow);
  }

  @Transactional
  public void unfollow(UUID followerId, UUID followedId) {
    followRepository.findByFollowerIdAndFollowedId(followerId, followedId).ifPresent(followRepository::delete);
  }

  @Transactional
  public void mute(UUID muterId, UUID mutedId) {
    getRequiredById(muterId);
    getRequiredById(mutedId);
    jdbcTemplate.update(
        """
        INSERT INTO mutes (id, muter_id, muted_id, created_at)
        VALUES (?, ?, ?, now())
        ON CONFLICT (muter_id, muted_id) DO NOTHING
        """,
        UUID.randomUUID(),
        muterId,
        mutedId);
  }

  @Transactional
  public void unmute(UUID muterId, UUID mutedId) {
    jdbcTemplate.update("DELETE FROM mutes WHERE muter_id = ? AND muted_id = ?", muterId, mutedId);
  }

  @Transactional
  public void hidePost(UUID userId, UUID postId) {
    getRequiredById(userId);
    jdbcTemplate.update(
        """
        INSERT INTO post_hides (id, user_id, post_id, created_at)
        VALUES (?, ?, ?, now())
        ON CONFLICT (user_id, post_id) DO NOTHING
        """,
        UUID.randomUUID(),
        userId,
        postId);
  }

  public List<UUID> listFollowerIds(UUID followedId) {
    return followRepository.findFollowers(followedId).stream().map(FollowEntity::getFollowerId).toList();
  }

  public List<UUID> listFollowingIds(UUID followerId) {
    return followRepository.findFollowing(followerId).stream().map(FollowEntity::getFollowedId).toList();
  }

  @Transactional
  public UserEntity findOrCreateRemoteActor(String actorUrl) {
    URI uri = URI.create(actorUrl);
    String host = uri.getHost();
    String[] parts = uri.getPath().split("/");
    String parsedUsername = parts.length == 0 ? "remote" : parts[parts.length - 1];
    final String username = parsedUsername.isBlank() ? "remote" : parsedUsername;

    Optional<UserEntity> existing = userRepository.findByUsernameAndDomain(username, host);
    if (existing.isPresent()) {
      return existing.get();
    }

    UserEntity user = new UserEntity();
    user.setId(UUID.randomUUID());
    user.setUsername(username);
    user.setDomain(host);
    user.setEmail(null);
    user.setPasswordHash(null);
    user.setDisplayName(username);
    user.setBio("");
    user.setAvatarUrl("");
    user.setPublicKey("");
    user.setPrivateKey(null);
    user.setCreatedAt(Instant.now(clock));
    return userRepository.save(user);
  }
}
