package com.gnusocial.next.api.config;

import org.springframework.boot.context.properties.ConfigurationProperties;

@ConfigurationProperties(prefix = "app")
public class AppProperties {
  private String baseUrl = "http://localhost";
  private String domain = "localhost";
  private boolean requireHttps = false;

  public String getBaseUrl() {
    return baseUrl;
  }

  public void setBaseUrl(String baseUrl) {
    this.baseUrl = baseUrl;
  }

  public String getDomain() {
    return domain;
  }

  public void setDomain(String domain) {
    this.domain = domain;
  }

  public boolean isRequireHttps() {
    return requireHttps;
  }

  public void setRequireHttps(boolean requireHttps) {
    this.requireHttps = requireHttps;
  }
}

