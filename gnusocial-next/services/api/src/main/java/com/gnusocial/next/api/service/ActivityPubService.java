package com.gnusocial.next.api.service;

import com.fasterxml.jackson.core.JsonProcessingException;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.gnusocial.next.api.config.AppProperties;
import com.gnusocial.next.api.dto.PostResponse;
import com.gnusocial.next.api.model.UserEntity;
import java.sql.Timestamp;
import java.time.Clock;
import java.time.Instant;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.Optional;
import java.util.UUID;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

@Service
public class ActivityPubService {
  private final AppProperties appProperties;
  private final UserService userService;
  private final PostService postService;
  private final JdbcTemplate jdbcTemplate;
  private final ObjectMapper objectMapper;
  private final Clock clock;

  public ActivityPubService(
      AppProperties appProperties,
      UserService userService,
      PostService postService,
      JdbcTemplate jdbcTemplate,
      ObjectMapper objectMapper,
      Clock clock) {
    this.appProperties = appProperties;
    this.userService = userService;
    this.postService = postService;
    this.jdbcTemplate = jdbcTemplate;
    this.objectMapper = objectMapper;
    this.clock = clock;
  }

  public Map<String, Object> webfinger(String resource) {
    if (resource == null || !resource.startsWith("acct:")) {
      throw new IllegalArgumentException("resource must start with acct:");
    }

    String acct = resource.substring("acct:".length());
    String username = acct;
    int at = acct.indexOf("@");
    if (at > -1) {
      username = acct.substring(0, at);
    }

    UserEntity user = userService.getRequiredLocalByUsername(username);
    Map<String, Object> response = new HashMap<>();
    response.put("subject", "acct:" + user.getUsername() + "@" + appProperties.getDomain());
    response.put(
        "links",
        List.of(
            Map.of(
                "rel", "self",
                "type", "application/activity+json",
                "href", appProperties.getBaseUrl() + "/users/" + user.getUsername())));
    return response;
  }

  public Map<String, Object> nodeInfoWellKnown() {
    return Map.of(
        "links",
        List.of(
            Map.of(
                "rel", "http://nodeinfo.diaspora.software/ns/schema/2.1",
                "href", appProperties.getBaseUrl() + "/nodeinfo/2.1")));
  }

  public Map<String, Object> nodeInfo() {
    return Map.of(
        "version", "2.1",
        "software", Map.of("name", "gnusocial-next", "version", "0.1.0"),
        "protocols", List.of("activitypub"),
        "services", Map.of("inbound", List.of(), "outbound", List.of()),
        "openRegistrations", true,
        "usage", Map.of("users", Map.of("total", localUserCount()), "localPosts", localPostCount()),
        "metadata", Map.of("queue", "redis-streams"));
  }

  public String hostMeta() {
    return
        """
        <?xml version="1.0" encoding="UTF-8"?>
        <XRD xmlns="http://docs.oasis-open.org/ns/xri/xrd-1.0">
          <Link rel="lrdd" type="application/xrd+xml" template="%s/.well-known/webfinger?resource={uri}"/>
        </XRD>
        """
            .formatted(appProperties.getBaseUrl());
  }

  public Map<String, Object> actorDocument(String username) {
    UserEntity user = userService.getRequiredLocalByUsername(username);
    String userUrl = appProperties.getBaseUrl() + "/users/" + user.getUsername();
    Map<String, Object> doc = new HashMap<>();
    doc.put("@context", List.of("https://www.w3.org/ns/activitystreams", "https://w3id.org/security/v1"));
    doc.put("id", userUrl);
    doc.put("type", "Person");
    doc.put("preferredUsername", user.getUsername());
    doc.put("name", user.getDisplayName());
    doc.put("summary", user.getBio());
    doc.put("inbox", userUrl + "/inbox");
    doc.put("outbox", userUrl + "/outbox");
    doc.put("followers", userUrl + "/followers");
    doc.put("following", userUrl + "/following");
    doc.put("publicKey",
        Map.of(
            "id", userUrl + "#main-key",
            "owner", userUrl,
            "publicKeyPem", user.getPublicKey()));
    if (user.getAvatarUrl() != null && !user.getAvatarUrl().isBlank()) {
      doc.put("icon", Map.of("type", "Image", "mediaType", "image/jpeg", "url", user.getAvatarUrl()));
    }
    return doc;
  }

  public Map<String, Object> followers(String username) {
    UserEntity user = userService.getRequiredLocalByUsername(username);
    List<String> items =
        userService.listFollowerIds(user.getId()).stream()
            .map(id -> appProperties.getBaseUrl() + "/users/id/" + id)
            .toList();
    return orderedCollection(user.getUsername() + "/followers", items);
  }

  public Map<String, Object> following(String username) {
    UserEntity user = userService.getRequiredLocalByUsername(username);
    List<String> items =
        userService.listFollowingIds(user.getId()).stream()
            .map(id -> appProperties.getBaseUrl() + "/users/id/" + id)
            .toList();
    return orderedCollection(user.getUsername() + "/following", items);
  }

  public Map<String, Object> userOutbox(String username, int limit) {
    UserEntity user = userService.getRequiredLocalByUsername(username);
    List<Map<String, Object>> items =
        postService.getUserOutbox(user.getId(), limit).stream().map(this::toCreateActivity).toList();
    return orderedCollection(user.getUsername() + "/outbox", items);
  }

  public Map<String, Object> userInbox(String username) {
    UserEntity user = userService.getRequiredLocalByUsername(username);
    return orderedCollection(user.getUsername() + "/inbox", List.of());
  }

  public Map<String, Object> outbox(int limit) {
    List<Map<String, Object>> items =
        postService.getPublicTimeline(limit).stream().map(this::toCreateActivity).toList();
    return orderedCollection("outbox", items);
  }

  @Transactional
  public void handleInbox(Map<String, Object> payload) {
    String type = value(payload.get("type"));
    if (type == null) {
      throw new IllegalArgumentException("Activity missing type");
    }

    String actor = value(payload.get("actor"));
    UserEntity actorUser = actor == null ? null : userService.findOrCreateRemoteActor(actor);

    UUID objectId = null;
    if ("Create".equalsIgnoreCase(type) && payload.get("object") instanceof Map<?, ?> objectRaw) {
      @SuppressWarnings("unchecked")
      Map<String, Object> object = (Map<String, Object>) objectRaw;
      Optional<PostResponse> post = postService.createRemotePostIfMissing(actor, object);
      if (post.isPresent()) {
        objectId = post.get().id();
      }
    }

    storeActivity(type, actorUser == null ? null : actorUser.getId(), objectId, payload);
  }

  private Map<String, Object> orderedCollection(String suffix, Object items) {
    String id = appProperties.getBaseUrl() + "/users/" + suffix;
    return Map.of(
        "@context", "https://www.w3.org/ns/activitystreams",
        "id", id,
        "type", "OrderedCollection",
        "totalItems", items instanceof List<?> list ? list.size() : 0,
        "orderedItems", items);
  }

  private Map<String, Object> toCreateActivity(PostResponse post) {
    return Map.of(
        "id", appProperties.getBaseUrl() + "/activities/" + post.id(),
        "type", "Create",
        "actor", appProperties.getBaseUrl() + "/users/id/" + post.authorId(),
        "object",
            Map.of(
                "id", post.federationId(),
                "type", "Note",
                "content", post.content(),
                "published", post.createdAt().toString()));
  }

  private long localUserCount() {
    Long count = jdbcTemplate.queryForObject("SELECT COUNT(*) FROM users WHERE domain IS NULL", Long.class);
    return count == null ? 0L : count;
  }

  private long localPostCount() {
    Long count = jdbcTemplate.queryForObject("SELECT COUNT(*) FROM posts", Long.class);
    return count == null ? 0L : count;
  }

  private void storeActivity(String type, UUID actorId, UUID objectId, Map<String, Object> payload) {
    try {
      jdbcTemplate.update(
          """
          INSERT INTO activities (id, type, actor_id, object_id, payload, created_at)
          VALUES (?, ?, ?, ?, CAST(? AS jsonb), ?)
          """,
          UUID.randomUUID(),
          type,
          actorId,
          objectId,
          objectMapper.writeValueAsString(payload),
          Timestamp.from(Instant.now(clock)));
    } catch (JsonProcessingException ex) {
      throw new IllegalStateException("Unable to store activity", ex);
    }
  }

  private String value(Object raw) {
    return raw == null ? null : String.valueOf(raw);
  }
}
