import TicketForm from './TicketForm';
import type { Service } from './ServiceCatalog';

export default function ServiceForm({ service }: { service: Service }) {
  return (
    <div className="mx-auto max-w-2xl space-y-4 p-4">
      <h2 className="text-2xl font-semibold">{service.title}</h2>
      <p>{service.description}</p>
      <TicketForm
        initial={{ title: service.title, category: service.category }}
        hideTitle
        hideCategory
      />
    </div>
  );
}
