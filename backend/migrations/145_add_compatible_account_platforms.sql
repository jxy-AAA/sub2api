DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'accounts_platform_allowed'
    ) THEN
        ALTER TABLE accounts DROP CONSTRAINT accounts_platform_allowed;
    END IF;
END $$;

ALTER TABLE accounts
ADD CONSTRAINT accounts_platform_allowed
CHECK (
    platform IN (
        'anthropic',
        'openai',
        'gemini',
        'antigravity',
        'openai_compatible',
        'anthropic_compatible'
    )
);
