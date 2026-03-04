package com.gnusocial.next.api.controller;

import com.gnusocial.next.api.dto.CreatePostRequest;
import com.gnusocial.next.api.dto.PostResponse;
import com.gnusocial.next.api.service.PostService;
import jakarta.validation.Valid;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
@RequestMapping("/api/v1")
public class PostController {
  private final PostService postService;

  public PostController(PostService postService) {
    this.postService = postService;
  }

  @PostMapping("/status")
  public PostResponse create(@Valid @RequestBody CreatePostRequest request) {
    return postService.createPost(request);
  }
}

