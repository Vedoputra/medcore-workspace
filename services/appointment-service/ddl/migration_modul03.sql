-- Modul 03: izinkan status PENDING_MEDICATION_SUBSTITUTION pada database yang
-- sudah berjalan (init.sql hanya dieksekusi saat volume baru dibuat).
ALTER TABLE appointments
    DROP CONSTRAINT IF EXISTS appointments_consultation_status_check,
    ALTER COLUMN consultation_status TYPE VARCHAR(40),
    ADD CONSTRAINT appointments_consultation_status_check CHECK (consultation_status IN (
        'SCHEDULED',
        'IN_PROGRESS',
        'COMPLETED',
        'CANCELLED',
        'PENDING_MEDICATION_SUBSTITUTION'
    ));
