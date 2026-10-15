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


-- Training (教育) for the Japanese PrivacyMark: yearly training plans, the
-- documents used as materials, and one record per participant. Attendance is
-- derived from signatures on the materials within the plan period, or entered
-- by hand (attended_on) for classroom training.
CREATE TABLE training_plans (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL
        REFERENCES organizations(id) ON DELETE CASCADE,
    fiscal_year INTEGER NOT NULL,
    name TEXT NOT NULL,
    kind TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    method TEXT NOT NULL,
    instructor TEXT NOT NULL DEFAULT '',
    starts_on DATE NOT NULL,
    ends_on DATE NOT NULL,
    time_zone TEXT NOT NULL,
    test_required BOOLEAN NOT NULL DEFAULT FALSE,
    pass_score INTEGER,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL,
    CONSTRAINT training_plans_kind_check CHECK (
        kind IN ('PERIODIC', 'ONBOARDING', 'ROLE_BASED', 'AD_HOC')
    ),
    CONSTRAINT training_plans_method_check CHECK (
        method IN ('CLASSROOM', 'ONLINE', 'E_LEARNING', 'MATERIALS')
    ),
    CONSTRAINT training_plans_period_check CHECK (starts_on <= ends_on),
    CONSTRAINT training_plans_fiscal_year_check CHECK (
        fiscal_year BETWEEN 2000 AND 2100
    ),
    CONSTRAINT training_plans_pass_score_check CHECK (
        pass_score IS NULL OR pass_score BETWEEN 0 AND 100
    )
);

CREATE INDEX training_plans_organization_id_idx
    ON training_plans (organization_id);

CREATE TABLE training_plan_materials (
    training_plan_id TEXT NOT NULL
        REFERENCES training_plans(id) ON DELETE CASCADE,
    document_id TEXT NOT NULL
        REFERENCES documents(id) ON DELETE CASCADE,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL
        REFERENCES organizations(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    PRIMARY KEY (training_plan_id, document_id)
);

CREATE INDEX training_plan_materials_document_id_idx
    ON training_plan_materials (document_id);

CREATE INDEX training_plan_materials_organization_id_idx
    ON training_plan_materials (organization_id);

CREATE TABLE training_participants (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL
        REFERENCES organizations(id) ON DELETE CASCADE,
    training_plan_id TEXT NOT NULL
        REFERENCES training_plans(id) ON DELETE CASCADE,
    profile_id TEXT NOT NULL
        REFERENCES iam_membership_profiles(id) ON DELETE CASCADE,
    attended_on DATE,
    test_score INTEGER,
    test_passed BOOLEAN,
    note TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL,
    CONSTRAINT training_participants_plan_profile_key
        UNIQUE (training_plan_id, profile_id),
    CONSTRAINT training_participants_test_score_check CHECK (
        test_score IS NULL OR test_score BETWEEN 0 AND 100
    )
);

CREATE INDEX training_participants_profile_id_idx
    ON training_participants (profile_id);

CREATE INDEX training_participants_organization_id_idx
    ON training_participants (organization_id);
