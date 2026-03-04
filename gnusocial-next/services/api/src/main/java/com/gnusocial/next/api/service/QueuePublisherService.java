package com.gnusocial.next.api.service;

import com.fasterxml.jackson.core.JsonProcessingException;
import com.fasterxml.jackson.databind.ObjectMapper;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.data.redis.connection.stream.MapRecord;
import org.springframework.data.redis.connection.stream.StreamRecords;
import org.springframework.data.redis.core.StringRedisTemplate;
import org.springframework.stereotype.Service;

@Service
public class QueuePublisherService {
  private final StringRedisTemplate redisTemplate;
  private final ObjectMapper objectMapper;
  private final String baseUrl;

  public QueuePublisherService(
      StringRedisTemplate redisTemplate,
      ObjectMapper objectMapper,
      @Value("${app.base-url:http://localhost}") String baseUrl) {
    this.redisTemplate = redisTemplate;
    this.objectMapper = objectMapper;
    this.baseUrl = baseUrl;
  }

  public void publishTimelineEvent(String postId, String authorId) {
    Map<String, String> payload = new HashMap<>();
    payload.put("post_id", postId);
    payload.put("author_id", authorId);
    add("timeline_events", payload);
  }

  public void publishFederationCreate(
      String actorUrl, Map<String, Object> object, List<String> targets, String activityId) {
    Map<String, Object> payload = new HashMap<>();
    payload.put("activity", "Create");
    payload.put("id", activityId);
    payload.put("actor", actorUrl);
    payload.put("object", object);
    payload.put("targets", targets);
    payload.put("origin", baseUrl);
    Map<String, String> streamPayload = new HashMap<>();
    streamPayload.put("payload", writeJson(payload));
    streamPayload.put("attempt", "0");
    add("federation_delivery", streamPayload);
  }

  public void publishMediaProcessing(String mediaId, String sourcePath) {
    Map<String, String> payload = new HashMap<>();
    payload.put("media_id", mediaId);
    payload.put("source_path", sourcePath);
    add("media_processing", payload);
  }

  private void add(String stream, Map<String, String> payload) {
    MapRecord<String, String, String> record = StreamRecords.mapBacked(payload).withStreamKey(stream);
    redisTemplate.opsForStream().add(record);
  }

  private String writeJson(Map<String, Object> payload) {
    try {
      return objectMapper.writeValueAsString(payload);
    } catch (JsonProcessingException ex) {
      throw new IllegalStateException("Unable to serialize stream payload", ex);
    }
  }
}

