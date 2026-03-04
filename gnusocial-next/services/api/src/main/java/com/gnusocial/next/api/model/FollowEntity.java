package com.gnusocial.next.api.model;

import jakarta.persistence.Column;
import jakarta.persistence.Entity;
import jakarta.persistence.Id;
import jakarta.persistence.Table;
import java.time.Instant;
import java.util.UUID;

@Entity
@Table(name = "follows")
public class FollowEntity {
  @Id
  private UUID id;

  @Column(name = "follower_id", nullable = false)
  private UUID followerId;

  @Column(name = "followed_id", nullable = false)
  private UUID followedId;

  @Column(name = "created_at", nullable = false)
  private Instant createdAt;

  public UUID getId() {
    return id;
  }

  public void setId(UUID id) {
    this.id = id;
  }

  public UUID getFollowerId() {
    return followerId;
  }

  public void setFollowerId(UUID followerId) {
    this.followerId = followerId;
  }

  public UUID getFollowedId() {
    return followedId;
  }

  public void setFollowedId(UUID followedId) {
    this.followedId = followedId;
  }

  public Instant getCreatedAt() {
    return createdAt;
  }

  public void setCreatedAt(Instant createdAt) {
    this.createdAt = createdAt;
  }
}

