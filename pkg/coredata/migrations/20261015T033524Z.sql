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


-- Japanese PrivacyMark risk analysis (リスク分析表) items that the risk does
-- not carry: the lifecycle stages of the personal information the risk
-- applies to, and the register entries (processing activities) it was
-- identified for. Kept in their own tables so the upstream risks table stays
-- as is.
CREATE TABLE risk_privacy_details (
    risk_id TEXT PRIMARY KEY
        REFERENCES risks(id) ON DELETE CASCADE,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL
        REFERENCES organizations(id) ON DELETE CASCADE,
    lifecycle_stages TEXT[] NOT NULL DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL,
    CONSTRAINT risk_privacy_details_lifecycle_stages_check CHECK (
        lifecycle_stages <@ ARRAY[
            'COLLECTION',
            'TRANSFER',
            'USE',
            'STORAGE',
            'DISPOSAL'
        ]::TEXT[]
    )
);

CREATE TABLE risk_processing_activities (
    risk_id TEXT NOT NULL
        REFERENCES risks(id) ON DELETE CASCADE,
    processing_activity_id TEXT NOT NULL
        REFERENCES processing_activities(id) ON DELETE CASCADE,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL
        REFERENCES organizations(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    PRIMARY KEY (risk_id, processing_activity_id)
);

CREATE INDEX risk_processing_activities_processing_activity_id_idx
    ON risk_processing_activities (processing_activity_id);

CREATE INDEX risk_processing_activities_organization_id_idx
    ON risk_processing_activities (organization_id);

CREATE INDEX risk_privacy_details_organization_id_idx
    ON risk_privacy_details (organization_id);
