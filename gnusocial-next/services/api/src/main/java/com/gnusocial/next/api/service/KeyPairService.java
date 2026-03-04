package com.gnusocial.next.api.service;

import java.security.KeyPair;
import java.security.KeyPairGenerator;
import java.security.NoSuchAlgorithmException;
import java.util.Base64;
import org.springframework.stereotype.Service;

@Service
public class KeyPairService {

  public GeneratedKeys generate() {
    try {
      KeyPairGenerator generator = KeyPairGenerator.getInstance("RSA");
      generator.initialize(2048);
      KeyPair pair = generator.generateKeyPair();
      return new GeneratedKeys(toPem("PUBLIC KEY", pair.getPublic().getEncoded()), toPem("PRIVATE KEY", pair.getPrivate().getEncoded()));
    } catch (NoSuchAlgorithmException ex) {
      throw new IllegalStateException("Unable to generate key pair", ex);
    }
  }

  private String toPem(String type, byte[] bytes) {
    String encoded = Base64.getMimeEncoder(64, "\n".getBytes()).encodeToString(bytes);
    return "-----BEGIN " + type + "-----\n" + encoded + "\n-----END " + type + "-----";
  }

  public record GeneratedKeys(String publicKey, String privateKey) {}
}

