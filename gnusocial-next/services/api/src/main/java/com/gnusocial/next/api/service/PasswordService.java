package com.gnusocial.next.api.service;

import org.springframework.security.crypto.argon2.Argon2PasswordEncoder;
import org.springframework.stereotype.Service;

@Service
public class PasswordService {
  private final Argon2PasswordEncoder encoder;

  public PasswordService(Argon2PasswordEncoder encoder) {
    this.encoder = encoder;
  }

  public String hash(String rawPassword) {
    return encoder.encode(rawPassword);
  }

  public boolean matches(String rawPassword, String encodedPassword) {
    return encoder.matches(rawPassword, encodedPassword);
  }
}

