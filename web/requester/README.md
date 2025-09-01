# Requester Portal

React portal for end users to submit and view helpdesk tickets.

## Development

```bash
npm install
npm run dev
```

## Styling

This app uses [Tailwind CSS](https://tailwindcss.com) for utility-first,
responsive styling. Tailwind is configured in `tailwind.config.ts` and
loaded through Vite.

### Build

```bash
npm run build
```

### Customization

- Edit `tailwind.config.ts` to extend themes or adjust breakpoints.
- Add shared styles in `src/index.css` using `@apply` with utility classes.
- Use Tailwind classes in components to control layout, spacing, and
  typography across breakpoints.

## Environment Variables

- `VITE_API_BASE` – Base URL for API (default `/api`).
- `VITE_OIDC_AUTHORITY` – OIDC provider URL.
- `VITE_OIDC_CLIENT_ID` – OIDC client ID.

The app supports ticket creation, listing, commenting, a basic knowledge base, and service catalog forms.
