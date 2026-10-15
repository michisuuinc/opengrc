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

-- Japanese PrivacyMark record of requests from individuals (本人からの請求の
-- 記録): how and when a request was received, who handled it, how the
-- individual was verified, the wait for missing documents, and the decision
-- and answer. Kept in its own table so the upstream rights_requests table
-- stays as is.
CREATE TABLE rights_request_privacy_details (
    rights_request_id TEXT PRIMARY KEY
        REFERENCES rights_requests(id) ON DELETE CASCADE,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL
        REFERENCES organizations(id) ON DELETE CASCADE,
    category TEXT,
    received_on DATE,
    receipt_channel TEXT,
    by_agent BOOLEAN NOT NULL DEFAULT FALSE,
    target_department TEXT NOT NULL DEFAULT '',
    -- The record stays when the handler or approver leaves.
    handler_profile_id TEXT
        REFERENCES iam_membership_profiles(id) ON DELETE SET NULL,
    verification_method TEXT,
    verified_on DATE,
    verification_note TEXT NOT NULL DEFAULT '',
    supplement_requested_on DATE,
    supplement_received_on DATE,
    decision TEXT,
    decision_reason TEXT NOT NULL DEFAULT '',
    answer_method TEXT,
    answered_on DATE,
    approver_profile_id TEXT
        REFERENCES iam_membership_profiles(id) ON DELETE SET NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL,
    CONSTRAINT rights_request_privacy_details_category_check CHECK (
        category IN (
            'PURPOSE_NOTIFICATION',
            'DISCLOSURE',
            'CORRECTION',
            'SUSPENSION',
            'THIRD_PARTY_PROVISION_STOP',
            'THIRD_PARTY_RECORD_DISCLOSURE',
            'COMPLAINT',
            'CONSULTATION'
        )
    ),
    CONSTRAINT rights_request_privacy_details_receipt_channel_check CHECK (
        receipt_channel IN ('POSTAL', 'EMAIL', 'WEB_FORM', 'PHONE', 'IN_PERSON', 'OTHER')
    ),
    CONSTRAINT rights_request_privacy_details_verification_method_check CHECK (
        verification_method IN ('ID_DOCUMENT', 'ACCOUNT', 'OTHER', 'NOT_REQUIRED')
    ),
    CONSTRAINT rights_request_privacy_details_decision_check CHECK (
        decision IN ('FULFILLED', 'PARTIALLY_FULFILLED', 'NOT_FULFILLED')
    ),
    CONSTRAINT rights_request_privacy_details_answer_method_check CHECK (
        answer_method IN ('POSTAL', 'EMAIL', 'ELECTRONIC_RECORD', 'PHONE', 'OTHER')
    )
);

CREATE INDEX rights_request_privacy_details_organization_id_idx
    ON rights_request_privacy_details (organization_id);

CREATE INDEX rights_request_privacy_details_handler_profile_id_idx
    ON rights_request_privacy_details (handler_profile_id);

CREATE INDEX rights_request_privacy_details_approver_profile_id_idx
    ON rights_request_privacy_details (approver_profile_id);
