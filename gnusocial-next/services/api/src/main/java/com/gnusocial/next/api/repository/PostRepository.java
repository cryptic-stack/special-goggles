package com.gnusocial.next.api.repository;

import com.gnusocial.next.api.model.PostEntity;
import java.util.List;
import java.util.Optional;
import java.util.UUID;
import org.springframework.data.domain.Pageable;
import org.springframework.data.jpa.repository.JpaRepository;
import org.springframework.data.jpa.repository.Query;
import org.springframework.data.repository.query.Param;

public interface PostRepository extends JpaRepository<PostEntity, UUID> {
  Optional<PostEntity> findByFederationId(String federationId);

  @Query("select p from PostEntity p where p.visibility = 'public' order by p.createdAt desc")
  List<PostEntity> findPublic(Pageable pageable);

  @Query("select p from PostEntity p where p.authorId = :authorId order by p.createdAt desc")
  List<PostEntity> findByAuthor(@Param("authorId") UUID authorId, Pageable pageable);
}

