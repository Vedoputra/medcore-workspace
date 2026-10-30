"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
const express_1 = __importDefault(require("express"));
const pg_1 = require("pg");
const app = (0, express_1.default)();
const port = process.env.PORT || 3000;
const databaseUrl = process.env.DATABASE_URL || 'postgres://usr_patient_svc:SecuredPassPatient2026!@localhost:5432/medcore_patient_db';
app.use(express_1.default.json());
const pool = new pg_1.Pool({
    connectionString: databaseUrl,
});
// Endpoint Healthcheck
app.get('/health', async (_req, res) => {
    try {
        await pool.query('SELECT 1');
        res.json({ status: 'UP', service: 'patient-service', database: 'connected' });
    }
    catch (err) {
        res.status(500).json({ status: 'DOWN', error: err.message });
    }
});
// Endpoint Ambil Seluruh Pasien
app.get('/api/v1/patients', async (_req, res) => {
    try {
        const result = await pool.query('SELECT * FROM patients ORDER BY created_at DESC');
        res.json({ success: true, data: result.rows });
    }
    catch (err) {
        res.status(500).json({ success: false, error: err.message });
    }
});
// Endpoint Detail Pasien
app.get('/api/v1/patients/:id', async (req, res) => {
    try {
        const result = await pool.query('SELECT * FROM patients WHERE patient_id = $1', [req.params.id]);
        if (result.rows.length === 0) {
            return res.status(404).json({ success: false, message: 'Patient not found' });
        }
        res.json({ success: true, data: result.rows[0] });
    }
    catch (err) {
        res.status(500).json({ success: false, error: err.message });
    }
});
const server = app.listen(port, () => {
    console.log(`[Patient Service] Running on port ${port}`);
});
// Graceful shutdown (Twelve-Factor App - Disposability)
process.on('SIGTERM', () => {
    console.log('[Patient Service] SIGTERM signal received: closing HTTP server');
    server.close(() => {
        pool.end();
        console.log('[Patient Service] HTTP server closed');
    });
});
