import TicketForm from './TicketForm';
import type { Service } from './ServiceCatalog';

export default function ServiceForm({ service }: { service: Service }) {
  return (
    <div>
      <h2>{service.title}</h2>
      <p>{service.description}</p>
      <TicketForm initial={{ title: service.title, category: service.category }} hideTitle hideCategory />
    </div>
  );
}
