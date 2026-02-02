ALTER TABLE custom_domains DROP CONSTRAINT custom_domains_domain_path_prefix_key;
ALTER TABLE custom_domains ADD CONSTRAINT custom_domains_domain_key UNIQUE (domain);
