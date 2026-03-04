package com.gnusocial.next.api.repository;

import com.gnusocial.next.api.model.FollowEntity;
import java.util.List;
import java.util.Optional;
import java.util.UUID;
import org.springframework.data.jpa.repository.JpaRepository;
import org.springframework.data.jpa.repository.Query;
import org.springframework.data.repository.query.Param;

public interface FollowRepository extends JpaRepository<FollowEntity, UUID> {
  Optional<FollowEntity> findByFollowerIdAndFollowedId(UUID followerId, UUID followedId);

  @Query("select f from FollowEntity f where f.followedId = :followedId")
  List<FollowEntity> findFollowers(@Param("followedId") UUID followedId);

  @Query("select f from FollowEntity f where f.followerId = :followerId")
  List<FollowEntity> findFollowing(@Param("followerId") UUID followerId);
}

