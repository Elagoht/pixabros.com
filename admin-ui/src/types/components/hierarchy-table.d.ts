interface HierarchyNode<T = Record<string, unknown>> {
  id: string;
  parentId: string | null;
  data: T;
}

interface HierarchyTableProps<T> {
  name: string;
  renderRow: (node: HierarchyNode<T>) => React.ReactNode;
  renderActions?: (node: HierarchyNode<T>) => React.ReactNode;
  saveButtonText?: string;
  emptyState?: React.ReactNode;
}
