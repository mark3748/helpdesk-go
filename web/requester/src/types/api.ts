export interface Ticket {
  id: string | number;
  number?: string | number;
  title?: string;
  status?: string;
  description?: string;
  category?: string;
  subcategory?: string;
  priority?: number;
  urgency?: number;
  requester_id?: string;
}

export interface Comment {
  id: string | number;
  body_md?: string;
  body?: string;
}

export interface Attachment {
  id: string | number;
  filename?: string;
  bytes?: number;
  content_type?: string;
}
