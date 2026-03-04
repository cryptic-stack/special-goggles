package com.gnusocial.next.api;

import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;
import org.springframework.boot.context.properties.ConfigurationPropertiesScan;

@SpringBootApplication
@ConfigurationPropertiesScan
public class GnuSocialNextApiApplication {

  public static void main(String[] args) {
    SpringApplication.run(GnuSocialNextApiApplication.class, args);
  }
}

