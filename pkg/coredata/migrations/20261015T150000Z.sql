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

-- Japanese PrivacyMark outsourcing evaluation (委託先評価表): the
-- evaluation checklist that the third-party risk assessment does not carry.
-- An assessment has at most one evaluation, and one row per checklist item
-- that was answered. Kept in their own tables so the upstream
-- third_party_risk_assessments table stays as is.
CREATE TABLE third_party_privacy_evaluations (
    risk_assessment_id TEXT PRIMARY KEY
        REFERENCES third_party_risk_assessments(id) ON DELETE CASCADE,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL
        REFERENCES organizations(id) ON DELETE CASCADE,
    kind TEXT,
    ad_hoc_reason TEXT NOT NULL DEFAULT '',
    entrusted_work TEXT NOT NULL DEFAULT '',
    personal_data TEXT NOT NULL DEFAULT '',
    storage_countries TEXT NOT NULL DEFAULT '',
    -- The evaluation stays when the evaluator or approver leaves.
    evaluator_profile_id TEXT
        REFERENCES iam_membership_profiles(id) ON DELETE SET NULL,
    evaluated_on DATE,
    judgement TEXT,
    conditions TEXT NOT NULL DEFAULT '',
    approver_profile_id TEXT
        REFERENCES iam_membership_profiles(id) ON DELETE SET NULL,
    approved_on DATE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL,
    CONSTRAINT third_party_privacy_evaluations_kind_check CHECK (
        kind IN ('NEW', 'PERIODIC', 'AD_HOC')
    ),
    CONSTRAINT third_party_privacy_evaluations_judgement_check CHECK (
        judgement IN ('APPROVED', 'CONDITIONALLY_APPROVED', 'REJECTED')
    )
);

CREATE INDEX third_party_privacy_evaluations_organization_id_idx
    ON third_party_privacy_evaluations (organization_id);

CREATE INDEX third_party_privacy_evaluations_evaluator_profile_id_idx
    ON third_party_privacy_evaluations (evaluator_profile_id);

CREATE INDEX third_party_privacy_evaluations_approver_profile_id_idx
    ON third_party_privacy_evaluations (approver_profile_id);

CREATE TABLE third_party_privacy_evaluation_items (
    risk_assessment_id TEXT NOT NULL
        REFERENCES third_party_privacy_evaluations(risk_assessment_id)
        ON DELETE CASCADE,
    item TEXT NOT NULL,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL
        REFERENCES organizations(id) ON DELETE CASCADE,
    result TEXT,
    note TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    PRIMARY KEY (risk_assessment_id, item),
    CONSTRAINT third_party_privacy_evaluation_items_item_check CHECK (
        item IN (
            'CERTIFICATION',
            'MANAGEMENT_STRUCTURE',
            'RULES',
            'EMPLOYEE_TRAINING',
            'PHYSICAL_SECURITY',
            'ACCESS_CONTROL',
            'TECHNICAL_SECURITY',
            'SUBCONTRACTING',
            'INCIDENT_RESPONSE',
            'PAST_INCIDENTS',
            'RETURN_AND_DELETION',
            'FOREIGN_STORAGE',
            'CONTRACT'
        )
    ),
    CONSTRAINT third_party_privacy_evaluation_items_result_check CHECK (
        result IN ('SUITABLE', 'UNSUITABLE', 'NOT_APPLICABLE')
    )
);

CREATE INDEX third_party_privacy_evaluation_items_organization_id_idx
    ON third_party_privacy_evaluation_items (organization_id);
