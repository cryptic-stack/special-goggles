package com.gnusocial.next.api.controller;

import com.gnusocial.next.api.service.ActivityPubService;
import java.util.Map;
import org.springframework.http.MediaType;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class ActivityPubController {
  private final ActivityPubService activityPubService;

  public ActivityPubController(ActivityPubService activityPubService) {
    this.activityPubService = activityPubService;
  }

  @GetMapping("/.well-known/webfinger")
  public Map<String, Object> webfinger(@RequestParam("resource") String resource) {
    return activityPubService.webfinger(resource);
  }

  @GetMapping("/.well-known/nodeinfo")
  public Map<String, Object> nodeInfoDiscovery() {
    return activityPubService.nodeInfoWellKnown();
  }

  @GetMapping(value = "/.well-known/host-meta", produces = MediaType.APPLICATION_XML_VALUE)
  public String hostMeta() {
    return activityPubService.hostMeta();
  }

  @GetMapping("/nodeinfo/2.1")
  public Map<String, Object> nodeInfo() {
    return activityPubService.nodeInfo();
  }

  @GetMapping("/users/{username}")
  public Map<String, Object> actor(@PathVariable String username) {
    return activityPubService.actorDocument(username);
  }

  @GetMapping("/users/{username}/followers")
  public Map<String, Object> followers(@PathVariable String username) {
    return activityPubService.followers(username);
  }

  @GetMapping("/users/{username}/following")
  public Map<String, Object> following(@PathVariable String username) {
    return activityPubService.following(username);
  }

  @GetMapping("/users/{username}/outbox")
  public Map<String, Object> userOutbox(
      @PathVariable String username, @RequestParam(name = "limit", defaultValue = "20") int limit) {
    return activityPubService.userOutbox(username, limit);
  }

  @GetMapping("/users/{username}/inbox")
  public Map<String, Object> userInbox(@PathVariable String username) {
    return activityPubService.userInbox(username);
  }

  @PostMapping("/users/{username}/inbox")
  public ResponseEntity<Map<String, String>> userInboxPost(
      @PathVariable String username, @RequestBody Map<String, Object> payload) {
    activityPubService.handleInbox(payload);
    return ResponseEntity.accepted().body(Map.of("status", "accepted", "inbox", username));
  }

  @PostMapping("/inbox")
  public ResponseEntity<Map<String, String>> inbox(@RequestBody Map<String, Object> payload) {
    activityPubService.handleInbox(payload);
    return ResponseEntity.accepted().body(Map.of("status", "accepted"));
  }

  @GetMapping("/outbox")
  public Map<String, Object> outbox(@RequestParam(name = "limit", defaultValue = "20") int limit) {
    return activityPubService.outbox(limit);
  }
}

