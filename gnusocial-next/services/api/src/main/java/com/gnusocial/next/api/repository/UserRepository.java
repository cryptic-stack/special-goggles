package com.gnusocial.next.api.repository;

import com.gnusocial.next.api.model.UserEntity;
import java.util.Optional;
import java.util.UUID;
import org.springframework.data.jpa.repository.JpaRepository;

public interface UserRepository extends JpaRepository<UserEntity, UUID> {
  Optional<UserEntity> findByUsernameAndDomain(String username, String domain);

  Optional<UserEntity> findByUsernameAndDomainIsNull(String username);

  Optional<UserEntity> findByEmail(String email);
}

