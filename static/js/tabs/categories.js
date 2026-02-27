// Categories tab — expand/collapse state helpers

export const categoriesState = {
  expandedCategories: {},
};

export function toggleCategory(expanded, index) {
  return { ...expanded, [index]: !expanded[index] };
}

export function isCategoryExpanded(expanded, index) {
  return !!expanded[index];
}
