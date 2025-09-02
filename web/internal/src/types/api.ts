export interface Ticket {
  id: string | number;
  number?: string | number;
  title?: string;
  status?: string;
  assignee_id?: string | null;
  priority?: number;
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

