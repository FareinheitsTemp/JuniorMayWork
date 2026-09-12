import SchemaMap from '@/components/SchemaMap';
import '@/styles/blocks/schema-map.scss';

// Спочатку ERD-мапа, нижче — Supabase-style data grid з page.jsx.
export default function DatabaseLayout({ children }) {
  return (
    <>
      <SchemaMap />
      <section className="schema__tables">
        <h2 className="page__title">Таблиці та записи</h2>
        {children}
      </section>
    </>
  );
}
