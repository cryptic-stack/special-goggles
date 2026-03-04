package com.gnusocial.next.api.controller;

import com.gnusocial.next.api.dto.PostResponse;
import com.gnusocial.next.api.service.TimelineService;
import java.util.List;
import java.util.UUID;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.RestController;

@RestController
@RequestMapping("/api/v1/timeline")
public class TimelineController {
  private final TimelineService timelineService;

  public TimelineController(TimelineService timelineService) {
    this.timelineService = timelineService;
  }

  @GetMapping("/home")
  public List<PostResponse> home(
      @RequestParam("userId") UUID userId,
      @RequestParam(name = "limit", defaultValue = "20") int limit,
      @RequestParam(name = "offset", defaultValue = "0") int offset) {
    return timelineService.getHomeTimeline(userId, limit, offset);
  }

  @GetMapping("/public")
  public List<PostResponse> publicTimeline(
      @RequestParam(name = "viewerId", required = false) UUID viewerId,
      @RequestParam(name = "limit", defaultValue = "20") int limit,
      @RequestParam(name = "offset", defaultValue = "0") int offset) {
    return timelineService.getPublicTimeline(viewerId, limit, offset);
  }
}

