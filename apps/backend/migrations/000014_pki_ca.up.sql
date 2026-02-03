CREATE TABLE pki_ca (
    name VARCHAR(50) PRIMARY KEY,
    cert_pem TEXT NOT NULL,
    key_pem TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TRIGGER update_pki_ca_updated_at
    BEFORE UPDATE ON pki_ca
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

COMMENT ON TABLE pki_ca IS 'Root CA used for agent mTLS';
