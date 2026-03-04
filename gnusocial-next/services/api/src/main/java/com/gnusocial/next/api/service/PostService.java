package com.gnusocial.next.api.service;

import com.fasterxml.jackson.core.JsonProcessingException;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.gnusocial.next.api.config.AppProperties;
import com.gnusocial.next.api.dto.CreatePostRequest;
import com.gnusocial.next.api.dto.PostResponse;
import com.gnusocial.next.api.dto.VisibilityType;
import com.gnusocial.next.api.model.PostEntity;
import com.gnusocial.next.api.model.UserEntity;
import com.gnusocial.next.api.repository.PostRepository;
import com.gnusocial.next.api.repository.UserRepository;
import java.sql.Timestamp;
import java.time.Clock;
import java.time.Instant;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.Optional;
import java.util.UUID;
import org.springframework.data.domain.PageRequest;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;
import org.springframework.transaction.support.TransactionSynchronization;
import org.springframework.transaction.support.TransactionSynchronizationManager;

@Service
public class PostService {
  private final PostRepository postRepository;
  private final UserRepository userRepository;
  private final UserService userService;
  private final QueuePublisherService queuePublisherService;
  private final JdbcTemplate jdbcTemplate;
  private final ObjectMapper objectMapper;
  private final AppProperties appProperties;
  private final Clock clock;

  public PostService(
      PostRepository postRepository,
      UserRepository userRepository,
      UserService userService,
      QueuePublisherService queuePublisherService,
      JdbcTemplate jdbcTemplate,
      ObjectMapper objectMapper,
      AppProperties appProperties,
      Clock clock) {
    this.postRepository = postRepository;
    this.userRepository = userRepository;
    this.userService = userService;
    this.queuePublisherService = queuePublisherService;
    this.jdbcTemplate = jdbcTemplate;
    this.objectMapper = objectMapper;
    this.appProperties = appProperties;
    this.clock = clock;
  }

  @Transactional
  public PostResponse createPost(CreatePostRequest request) {
    String visibility = normalizeVisibility(request.visibility());
    UserEntity author = userService.getRequiredById(request.authorId());
    Instant now = Instant.now(clock);

    PostEntity post = new PostEntity();
    post.setId(UUID.randomUUID());
    post.setAuthorId(author.getId());
    post.setContent(request.content());
    post.setVisibility(visibility);
    post.setCreatedAt(now);
    post.setReplyTo(request.replyTo());
    post.setFederationId(appProperties.getBaseUrl() + "/objects/" + post.getId());
    PostEntity saved = postRepository.saveAndFlush(post);

    addTimelineEntry(saved.getAuthorId(), saved.getId(), now);

    String actorUrl = actorUrl(author);
    Map<String, Object> object = buildNoteObject(saved, actorUrl);
    String activityId = appProperties.getBaseUrl() + "/activities/" + UUID.randomUUID();
    Map<String, Object> activityPayload = new HashMap<>();
    activityPayload.put("@context", "https://www.w3.org/ns/activitystreams");
    activityPayload.put("id", activityId);
    activityPayload.put("type", "Create");
    activityPayload.put("actor", actorUrl);
    activityPayload.put("object", object);
    storeActivity("Create", author.getId(), saved.getId(), activityPayload);

    List<String> targets = resolveRemoteInboxes(author.getId());
    runAfterCommit(
        () -> {
          queuePublisherService.publishTimelineEvent(saved.getId().toString(), saved.getAuthorId().toString());
          if (!targets.isEmpty()) {
            queuePublisherService.publishFederationCreate(actorUrl, object, targets, activityId);
          }
        });

    return toResponse(saved);
  }

  @Transactional
  public Optional<PostResponse> createRemotePostIfMissing(String actorUrl, Map<String, Object> object) {
    String federationId = stringValue(object.get("id"));
    if (federationId == null || federationId.isBlank()) {
      return Optional.empty();
    }
    if (postRepository.findByFederationId(federationId).isPresent()) {
      return Optional.empty();
    }

    UserEntity remoteUser = userService.findOrCreateRemoteActor(actorUrl);
    Instant now = Instant.now(clock);

    PostEntity post = new PostEntity();
    post.setId(UUID.randomUUID());
    post.setAuthorId(remoteUser.getId());
    post.setContent(stringValue(object.getOrDefault("content", "")));
    post.setVisibility(VisibilityType.PUBLIC);
    post.setCreatedAt(now);
    post.setReplyTo(null);
    post.setFederationId(federationId);

    PostEntity saved = postRepository.saveAndFlush(post);
    runAfterCommit(() -> queuePublisherService.publishTimelineEvent(saved.getId().toString(), saved.getAuthorId().toString()));
    return Optional.of(toResponse(saved));
  }

  public List<PostResponse> getPublicTimeline(int limit) {
    return postRepository.findPublic(PageRequest.of(0, sanitizeLimit(limit))).stream().map(this::toResponse).toList();
  }

  public List<PostResponse> getUserOutbox(UUID userId, int limit) {
    return postRepository.findByAuthor(userId, PageRequest.of(0, sanitizeLimit(limit))).stream().map(this::toResponse).toList();
  }

  public PostResponse toResponse(PostEntity post) {
    return new PostResponse(
        post.getId(),
        post.getAuthorId(),
        post.getContent(),
        post.getVisibility(),
        post.getReplyTo(),
        post.getFederationId(),
        post.getCreatedAt());
  }

  private String normalizeVisibility(String visibility) {
    String value = visibility == null ? VisibilityType.PUBLIC : visibility.toLowerCase();
    if (!value.equals(VisibilityType.PUBLIC)
        && !value.equals(VisibilityType.UNLISTED)
        && !value.equals(VisibilityType.FOLLOWERS)
        && !value.equals(VisibilityType.DIRECT)) {
      throw new IllegalArgumentException("Unsupported visibility: " + visibility);
    }
    return value;
  }

  private int sanitizeLimit(int limit) {
    if (limit < 1) {
      return 20;
    }
    return Math.min(limit, 100);
  }

  private void addTimelineEntry(UUID userId, UUID postId, Instant createdAt) {
    jdbcTemplate.update(
        """
        INSERT INTO timeline_entries (user_id, post_id, created_at)
        VALUES (?, ?, ?)
        ON CONFLICT (user_id, post_id) DO NOTHING
        """,
        userId,
        postId,
        Timestamp.from(createdAt));
  }

  private List<String> resolveRemoteInboxes(UUID followedAuthorId) {
    List<UUID> followerIds = userService.listFollowerIds(followedAuthorId);
    if (followerIds.isEmpty()) {
      return List.of();
    }

    List<String> targets = new ArrayList<>();
    for (UUID followerId : followerIds) {
      userRepository.findById(followerId).ifPresent(user -> {
        if (user.getDomain() != null && !user.getDomain().isBlank()) {
          targets.add("https://" + user.getDomain() + "/inbox");
        }
      });
    }
    return targets;
  }

  private String actorUrl(UserEntity author) {
    if (author.getDomain() == null || author.getDomain().isBlank()) {
      return appProperties.getBaseUrl() + "/users/" + author.getUsername();
    }
    return "https://" + author.getDomain() + "/users/" + author.getUsername();
  }

  private Map<String, Object> buildNoteObject(PostEntity post, String actorUrl) {
    Map<String, Object> object = new HashMap<>();
    object.put("id", post.getFederationId());
    object.put("type", "Note");
    object.put("attributedTo", actorUrl);
    object.put("content", post.getContent());
    object.put("published", post.getCreatedAt().toString());
    return object;
  }

  private void storeActivity(
      String type, UUID actorId, UUID objectId, Map<String, Object> activityPayload) {
    try {
      String payload = objectMapper.writeValueAsString(activityPayload);
      jdbcTemplate.update(
          """
          INSERT INTO activities (id, type, actor_id, object_id, payload, created_at)
          VALUES (?, ?, ?, ?, CAST(? AS jsonb), ?)
          """,
          UUID.randomUUID(),
          type,
          actorId,
          objectId,
          payload,
          Timestamp.from(Instant.now(clock)));
    } catch (JsonProcessingException ex) {
      throw new IllegalStateException("Unable to store activity", ex);
    }
  }

  private String stringValue(Object value) {
    return value == null ? null : String.valueOf(value);
  }

  private void runAfterCommit(Runnable action) {
    if (TransactionSynchronizationManager.isActualTransactionActive()) {
      TransactionSynchronizationManager.registerSynchronization(
          new TransactionSynchronization() {
            @Override
            public void afterCommit() {
              action.run();
            }
          });
      return;
    }
    action.run();
  }
}
