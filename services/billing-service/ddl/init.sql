CREATE TABLE invoices (
    invoice_id VARCHAR(36) PRIMARY KEY,
    patient_id VARCHAR(36) NOT NULL, -- Logical ID
    appointment_id VARCHAR(36) NOT NULL, -- Logical ID
    total_amount DECIMAL(12,2) NOT NULL,
    payment_status VARCHAR(20) DEFAULT 'UNPAID',
    issued_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_appt (appointment_id),
    INDEX idx_patient (patient_id)
) ENGINE=InnoDB;

CREATE TABLE invoice_items (
    item_id VARCHAR(36) PRIMARY KEY,
    invoice_id VARCHAR(36) NOT NULL,
    description VARCHAR(255) NOT NULL,
    amount DECIMAL(12,2) NOT NULL,
    FOREIGN KEY (invoice_id) REFERENCES invoices(invoice_id) ON DELETE
CASCADE
) ENGINE=InnoDB;

INSERT INTO invoices (invoice_id, patient_id, appointment_id,
total_amount, payment_status)
VALUES ('inv-0001', 'a0000000-0000-0000-0000-000000000001',
'b1111111-1111-1111-1111-111111111111', 150000.00, 'UNPAID');

INSERT INTO invoice_items (item_id, invoice_id, description, amount)
VALUES
('item-1', 'inv-0001', 'Jasa Konsultasi Spesialis Penyakit Dalam',
100000.00),
('item-2', 'inv-0001', 'Biaya Administrasi Rawat Jalan', 50000.00);