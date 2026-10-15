-- Copyright (c) 2026 MICHISUU Inc.
--
-- Permission is hereby granted, free of charge, to any person obtaining a copy
-- of this software and associated documentation files (the "Software"), to deal
-- in the Software without restriction, including without limitation the rights
-- to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
-- copies of the Software, and to permit persons to whom the Software is
-- furnished to do so, subject to the following conditions:
--
-- The above copyright notice and this permission notice shall be included in
-- all copies or substantial portions of the Software.
--
-- THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
-- IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
-- FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
-- AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
-- LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
-- OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
-- SOFTWARE.

-- Japanese PrivacyMark internal audit report (内部監査報告書) items that the
-- audit and finding do not carry: the scope, criteria, auditors, audited
-- departments and overall opinion of an audit, and the category and basis
-- of a finding. Kept in their own tables so the upstream audits and findings
-- tables stay as is.
CREATE TABLE audit_privacy_details (
    audit_id TEXT PRIMARY KEY
        REFERENCES audits(id) ON DELETE CASCADE,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL
        REFERENCES organizations(id) ON DELETE CASCADE,
    scope TEXT NOT NULL DEFAULT '',
    criteria TEXT NOT NULL DEFAULT '',
    auditee_departments TEXT[] NOT NULL DEFAULT '{}',
    external_auditors TEXT[] NOT NULL DEFAULT '{}',
    overall_opinion TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL
);

CREATE INDEX audit_privacy_details_organization_id_idx
    ON audit_privacy_details (organization_id);

CREATE TABLE audit_privacy_auditors (
    audit_id TEXT NOT NULL
        REFERENCES audits(id) ON DELETE CASCADE,
    profile_id TEXT NOT NULL
        REFERENCES iam_membership_profiles(id) ON DELETE CASCADE,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL
        REFERENCES organizations(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    PRIMARY KEY (audit_id, profile_id)
);

CREATE INDEX audit_privacy_auditors_profile_id_idx
    ON audit_privacy_auditors (profile_id);

CREATE INDEX audit_privacy_auditors_organization_id_idx
    ON audit_privacy_auditors (organization_id);

CREATE TABLE finding_privacy_details (
    finding_id TEXT PRIMARY KEY
        REFERENCES findings(id) ON DELETE CASCADE,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL
        REFERENCES organizations(id) ON DELETE CASCADE,
    category TEXT,
    basis TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL,
    CONSTRAINT finding_privacy_details_category_check CHECK (
        category IN (
            'NONCONFORMITY',
            'OBSERVATION',
            'IMPROVEMENT_OPPORTUNITY'
        )
    )
);

CREATE INDEX finding_privacy_details_organization_id_idx
    ON finding_privacy_details (organization_id);
