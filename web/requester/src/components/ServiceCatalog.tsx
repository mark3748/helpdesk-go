import { Link, Route, Routes } from 'react-router-dom';
import ServiceForm from './ServiceForm';

export interface Service {
  id: string;
  title: string;
  category: string;
  description: string;
}

const services: Service[] = [
  { id: 'it-support', title: 'IT Support Request', category: 'support', description: 'General IT help' },
  { id: 'access', title: 'Access Request', category: 'access', description: 'Request system access' },
  { id: 'hardware', title: 'Hardware Request', category: 'hardware', description: 'Request new hardware' },
];

export default function ServiceCatalog() {
  return (
    <Routes>
      <Route
        path=""
        element={
          <div className="mx-auto max-w-2xl space-y-4 p-4">
            <h2 className="text-2xl font-semibold">Service Catalog</h2>
            <ul className="space-y-2">
              {services.map(s => (
                <li key={s.id} className="border-b pb-2 last:border-b-0">
                  <Link className="text-blue-600 hover:underline" to={s.id}>
                    {s.title}
                  </Link>{' '}
                  - {s.description}
                </li>
              ))}
            </ul>
          </div>
        }
      />
      {services.map(s => (
        <Route key={s.id} path={s.id} element={<ServiceForm service={s} />} />
      ))}
    </Routes>
  );
}
