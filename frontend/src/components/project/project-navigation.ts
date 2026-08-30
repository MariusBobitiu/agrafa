export function projectIdFromSearchParams(searchParams: URLSearchParams): number | null {
  const value = searchParams.get("project_id");
  if (value === null || !/^[1-9]\d*$/.test(value)) {
    return null;
  }

  const projectId = Number(value);
  return Number.isSafeInteger(projectId) ? projectId : null;
}
