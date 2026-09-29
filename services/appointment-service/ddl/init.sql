CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE doctors (
    doctor_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    license_number VARCHAR(50) UNIQUE NOT NULL,
    doctor_name VARCHAR(150) NOT NULL,
    specialization VARCHAR(50) NOT NULL,
    is_active BOOLEAN DEFAULT TRUE
);

CREATE TABLE appointments (
    appointment_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    patient_id UUID NOT NULL,
    -- Referensi logis ke Patient Service (decoupled)
    doctor_id UUID REFERENCES doctors(doctor_id),
    schedule_time TIMESTAMP WITH TIME ZONE NOT NULL,
    queue_number INT NOT NULL,
    consultation_status VARCHAR(20) DEFAULT 'SCHEDULED'
        CHECK (
            consultation_status IN (
                'SCHEDULED',
                'IN_PROGRESS',
                'COMPLETED',
                'CANCELLED'
            )
        ),
    clinical_notes TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Seed Data Awal
INSERT INTO doctors (
    doctor_id,
    license_number,
    doctor_name,
    specialization
)
VALUES
    (
        'b0000000-0000-0000-0000-000000000001',
        'SIP-BALI-9901',
        'dr. Ketut Gede, Sp.PD',
        'Penyakit Dalam'
    ),
    (
        'b0000000-0000-0000-0000-000000000002',
        'SIP-BALI-9902',
        'dr. Putu Ayu, Sp.A',
        'Spesialis Anak'
    );