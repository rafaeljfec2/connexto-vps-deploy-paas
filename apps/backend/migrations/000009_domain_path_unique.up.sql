ALTER TABLE custom_domains DROP CONSTRAINT custom_domains_domain_key;
ALTER TABLE custom_domains ADD CONSTRAINT custom_domains_domain_path_prefix_key UNIQUE (domain, path_prefix);
