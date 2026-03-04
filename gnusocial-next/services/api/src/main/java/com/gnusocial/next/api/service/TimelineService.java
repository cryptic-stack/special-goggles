package com.gnusocial.next.api.service;

import com.gnusocial.next.api.dto.PostResponse;
import java.sql.ResultSet;
import java.sql.SQLException;
import java.util.List;
import java.util.UUID;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.jdbc.core.RowMapper;
import org.springframework.stereotype.Service;

@Service
public class TimelineService {
  private final JdbcTemplate jdbcTemplate;

  public TimelineService(JdbcTemplate jdbcTemplate) {
    this.jdbcTemplate = jdbcTemplate;
  }

  public List<PostResponse> getHomeTimeline(UUID userId, int limit, int offset) {
    int safeLimit = sanitizeLimit(limit);
    int safeOffset = Math.max(offset, 0);
    return jdbcTemplate.query(
        """
        SELECT p.id, p.author_id, p.content, p.visibility, p.reply_to, p.federation_id, p.created_at
        FROM timeline_entries te
        JOIN posts p ON p.id = te.post_id
        LEFT JOIN post_hides ph ON ph.post_id = p.id AND ph.user_id = ?
        LEFT JOIN mutes m ON m.muted_id = p.author_id AND m.muter_id = ?
        WHERE te.user_id = ?
          AND ph.post_id IS NULL
          AND m.muted_id IS NULL
        ORDER BY te.created_at DESC
        LIMIT ? OFFSET ?
        """,
        postMapper(),
        userId,
        userId,
        userId,
        safeLimit,
        safeOffset);
  }

  public List<PostResponse> getPublicTimeline(UUID viewerId, int limit, int offset) {
    int safeLimit = sanitizeLimit(limit);
    int safeOffset = Math.max(offset, 0);
    if (viewerId == null) {
      return jdbcTemplate.query(
          """
          SELECT p.id, p.author_id, p.content, p.visibility, p.reply_to, p.federation_id, p.created_at
          FROM posts p
          WHERE p.visibility = 'public'
          ORDER BY p.created_at DESC
          LIMIT ? OFFSET ?
          """,
          postMapper(),
          safeLimit,
          safeOffset);
    }

    return jdbcTemplate.query(
        """
        SELECT p.id, p.author_id, p.content, p.visibility, p.reply_to, p.federation_id, p.created_at
        FROM posts p
        LEFT JOIN post_hides ph ON ph.post_id = p.id AND ph.user_id = ?
        LEFT JOIN mutes m ON m.muted_id = p.author_id AND m.muter_id = ?
        WHERE p.visibility = 'public'
          AND ph.post_id IS NULL
          AND m.muted_id IS NULL
        ORDER BY p.created_at DESC
        LIMIT ? OFFSET ?
        """,
        postMapper(),
        viewerId,
        viewerId,
        safeLimit,
        safeOffset);
  }

  private RowMapper<PostResponse> postMapper() {
    return (rs, ignored) -> mapPost(rs);
  }

  private PostResponse mapPost(ResultSet rs) throws SQLException {
    return new PostResponse(
        rs.getObject("id", UUID.class),
        rs.getObject("author_id", UUID.class),
        rs.getString("content"),
        rs.getString("visibility"),
        rs.getObject("reply_to", UUID.class),
        rs.getString("federation_id"),
        rs.getTimestamp("created_at").toInstant());
  }

  private int sanitizeLimit(int limit) {
    if (limit < 1) {
      return 20;
    }
    return Math.min(limit, 100);
  }
}

