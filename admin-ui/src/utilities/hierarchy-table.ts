export const detectCycle = (
  items: Array<{ id: string; parentId: string | null }>,
  itemId: string,
  targetParentId: string | null,
): boolean => {
  if (targetParentId === null) {
    return false;
  }

  const map = new Map<string, string | null>();
  for (const item of items) {
    map.set(item.id, item.parentId);
  }

  let current: string | null = targetParentId;
  while (current !== null) {
    if (current === itemId) {
      return true;
    }
    current = map.get(current) ?? null;
  }

  return false;
};

export const getAllDescendantIds = (
  items: Array<{ id: string; parentId: string | null }>,
  parentId: string,
): Set<string> => {
  const result = new Set<string>();
  const stack = [parentId];
  while (stack.length > 0) {
    const current = stack.pop();
    if (!current) {
      continue;
    }
    for (const item of items) {
      if (item.parentId === current && !result.has(item.id)) {
        result.add(item.id);
        stack.push(item.id);
      }
    }
  }
  return result;
};

export const applyDrop = <T extends { id: string; parentId: string | null }>(
  items: T[],
  draggedId: string,
  targetId: string | null,
  position: "before" | "after" | "child",
): T[] => {
  if (targetId === null) {
    const descendants = getAllDescendantIds(items, draggedId);
    const movingIds = new Set([draggedId, ...descendants]);
    const moving: T[] = [];
    const remaining: T[] = [];

    for (const item of items) {
      if (movingIds.has(item.id)) {
        moving.push({ ...item });
      } else {
        remaining.push(item);
      }
    }

    if (moving.length > 0) {
      moving[0].parentId = null;
      return [...remaining, ...moving];
    }
    return items;
  }

  const descendants = getAllDescendantIds(items, draggedId);
  if (descendants.has(targetId)) {
    return items;
  }

  const movingIds = new Set([draggedId, ...descendants]);
  const moving: T[] = [];
  const remaining: T[] = [];

  for (const item of items) {
    if (movingIds.has(item.id)) {
      moving.push({ ...item });
    } else {
      remaining.push(item);
    }
  }

  if (moving.length === 0) {
    return items;
  }

  const targetItem = remaining.find((n) => n.id === targetId);
  if (!targetItem) {
    return items;
  }

  if (position === "child") {
    moving[0].parentId = targetId;
  } else {
    moving[0].parentId = targetItem.parentId;
  }

  const targetIdx = remaining.findIndex((n) => n.id === targetId);
  if (targetIdx === -1) {
    return [...remaining, ...moving];
  }

  if (position === "child") {
    let insertAt = targetIdx + 1;
    while (
      insertAt < remaining.length &&
      remaining[insertAt].parentId === targetId
    ) {
      insertAt++;
    }
    remaining.splice(insertAt, 0, ...moving);
  } else if (position === "before") {
    remaining.splice(targetIdx, 0, ...moving);
  } else {
    remaining.splice(targetIdx + 1, 0, ...moving);
  }

  return remaining;
};

export const getDepth = (
  items: Array<{ id: string; parentId: string | null }>,
  nodeId: string,
): number => {
  const map = new Map(items.map((n) => [n.id, n.parentId]));

  let depth = 0;
  let current = map.get(nodeId) ?? null;
  while (current !== null) {
    depth++;
    current = map.get(current) ?? null;
  }

  return depth;
};

export const hasChildren = (
  items: Array<{ id: string; parentId: string | null }>,
  nodeId: string,
): boolean => items.some((n) => n.parentId === nodeId);
