package com.gnusocial.next.api.model;

import jakarta.persistence.Column;
import jakarta.persistence.Entity;
import jakarta.persistence.Id;
import jakarta.persistence.Table;
import java.time.Instant;
import java.util.UUID;

@Entity
@Table(name = "posts")
public class PostEntity {
  @Id
  private UUID id;

  @Column(name = "author_id", nullable = false)
  private UUID authorId;

  @Column(nullable = false)
  private String content;

  @Column(nullable = false)
  private String visibility;

  @Column(name = "created_at", nullable = false)
  private Instant createdAt;

  @Column(name = "reply_to")
  private UUID replyTo;

  @Column(name = "federation_id")
  private String federationId;

  public UUID getId() {
    return id;
  }

  public void setId(UUID id) {
    this.id = id;
  }

  public UUID getAuthorId() {
    return authorId;
  }

  public void setAuthorId(UUID authorId) {
    this.authorId = authorId;
  }

  public String getContent() {
    return content;
  }

  public void setContent(String content) {
    this.content = content;
  }

  public String getVisibility() {
    return visibility;
  }

  public void setVisibility(String visibility) {
    this.visibility = visibility;
  }

  public Instant getCreatedAt() {
    return createdAt;
  }

  public void setCreatedAt(Instant createdAt) {
    this.createdAt = createdAt;
  }

  public UUID getReplyTo() {
    return replyTo;
  }

  public void setReplyTo(UUID replyTo) {
    this.replyTo = replyTo;
  }

  public String getFederationId() {
    return federationId;
  }

  public void setFederationId(String federationId) {
    this.federationId = federationId;
  }
}

