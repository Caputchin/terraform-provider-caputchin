data "caputchin_site_stats" "blog" {
  site_id = "site_abc123"
}

# Feed counters into your monitoring stack. The counters are lifetime
# totals; time-series breakdowns are not exposed via this data source.
output "blog_sessions_verified" {
  value = data.caputchin_site_stats.blog.sessions_server_verified
}

output "blog_rate_limit_rejections" {
  value = data.caputchin_site_stats.blog.rate_limit_rejections
}
