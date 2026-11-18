export type PromptFn = (message: string) => string | null;

// Destructive actions require typing a phrase so a stray OK click cannot delete data.
export function confirmByTyping(
  message: string,
  phrase: string,
  prompt: PromptFn = (text) => window.prompt(text),
): boolean {
  const answer = prompt(`${message}\n\nThis cannot be undone. Type ${phrase} to confirm.`);
  return answer?.trim() === phrase;
}
