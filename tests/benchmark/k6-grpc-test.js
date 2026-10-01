import grpc from 'k6/net/grpc';
import { check } from 'k6';

const client = new grpc.Client();
client.load(['../../proto'], 'pharmacy.proto');

export const options = {
  stages: [
    { duration: '10s', target: 50 },
    { duration: '20s', target: 200 },
    { duration: '10s', target: 0 },
  ],
};

export default function () {
  // Koneksi dibuka SEKALI per VU dan dipakai ulang (HTTP/2 multiplexed),
  // bukan connect/close di setiap iterasi (menyebabkan port exhaustion di Windows).
  if (__ITER === 0) {
    client.connect('localhost:50051', { plaintext: true });
  }

  const response = client.invoke('medcore.pharmacy.v1.PharmacyService/CheckDrugAvailability', {
    drug_code: 'MED-AMX-500',
    quantity_needed: 10,
  });

  check(response, {
    'status is OK': (r) => r && r.status === grpc.StatusOK,
    // k6 dapat mengembalikan nama field camelCase (isAvailable) atau snake_case (is_available)
    'has valid stock': (r) => r && r.message && (r.message.isAvailable === true || r.message.is_available === true),
  });
}