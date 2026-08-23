export default function ProductDetailPage({ params }: { params: { id: string } }) {
  return (
    <div>
      <h1>Product Detail</h1>
      <p>ID: {params.id}</p>
    </div>
  );
}
