db = db.getSiblingDB('medcore_pharmacy_db');

db.createCollection('drugs');
db.createCollection('prescriptions');

db.drugs.createIndex({ "drug_code": 1 }, { unique: true });
db.prescriptions.createIndex({ "appointment_id": 1 });

db.drugs.insertMany([
  {
    drug_code: "MED-AMX-500",
    name: "Amoxicillin 500mg",
    category: "Antibiotik",
    unit_price: 3500.0,
    total_stock: 500,
    batches: [
      { batch_no: "BATCH-2026A", stock: 200, exp_date: new Date("2028-01-01") },
      { batch_no: "BATCH-2026B", stock: 300, exp_date: new Date("2028-06-01") }
    ]
  },
  {
    drug_code: "MED-PCT-500",
    name: "Paracetamol 500mg",
    category: "Analgesik",
    unit_price: 1500.0,
    total_stock: 1000,
    batches: [
      { batch_no: "BATCH-PCT-01", stock: 1000, exp_date: new Date("2029-01-01") }
    ]
  }
]);

db.prescriptions.insertOne({
  prescription_id: "RX-2026-0001",
  appointment_id: "b1111111-1111-1111-1111-111111111111",
  patient_id: "a0000000-0000-0000-0000-000000000001",
  items: [
    { drug_code: "MED-AMX-500", quantity: 15, instructions: "3x1 sehari sesudah makan" },
    { drug_code: "MED-PCT-500", quantity: 10, instructions: "3x1 bila demam" }
  ],
  status: "ISSUED",
  created_at: new Date()
});