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
          <div>
            <h2>Service Catalog</h2>
            <ul>
              {services.map(s => (
                <li key={s.id}>
                  <Link to={s.id}>{s.title}</Link> - {s.description}
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
