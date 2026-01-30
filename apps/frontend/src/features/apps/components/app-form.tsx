import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { FormField } from "@/components/form-field";
import { useCreateApp } from "../hooks/use-apps";

export function AppForm() {
  const navigate = useNavigate();
  const createApp = useCreateApp();

  const [name, setName] = useState("");
  const [repositoryUrl, setRepositoryUrl] = useState("");
  const [branch, setBranch] = useState("main");

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    try {
      await createApp.mutateAsync({
        name,
        repositoryUrl,
        branch,
      });
      navigate("/");
    } catch (error) {
      console.error("Failed to create app:", error);
    }
  };

  const isValid = name.length >= 2 && repositoryUrl.includes("github.com");

  return (
    <Card className="max-w-lg mx-auto">
      <form onSubmit={handleSubmit}>
        <CardHeader>
          <CardTitle>Connect Repository</CardTitle>
          <CardDescription>
            Add a GitHub repository to deploy automatically on push.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <FormField
            label="Application Name"
            htmlFor="name"
            helper="Lowercase letters, numbers, and hyphens only."
            required
          >
            <Input
              id="name"
              placeholder="my-app"
              value={name}
              onChange={(e) =>
                setName(
                  e.target.value.toLowerCase().replaceAll(/[^a-z0-9-]/g, "-"),
                )
              }
              required
              minLength={2}
              maxLength={63}
            />
          </FormField>

          <FormField label="Repository URL" htmlFor="repository" required>
            <Input
              id="repository"
              placeholder="https://github.com/owner/repo"
              value={repositoryUrl}
              onChange={(e) => setRepositoryUrl(e.target.value)}
              required
            />
          </FormField>

          <FormField label="Branch" htmlFor="branch">
            <Input
              id="branch"
              placeholder="main"
              value={branch}
              onChange={(e) => setBranch(e.target.value)}
            />
          </FormField>
        </CardContent>
        <CardFooter className="flex justify-between">
          <Button type="button" variant="outline" onClick={() => navigate("/")}>
            Cancel
          </Button>
          <Button type="submit" disabled={!isValid || createApp.isPending}>
            {createApp.isPending ? "Creating..." : "Create Application"}
          </Button>
        </CardFooter>
      </form>
    </Card>
  );
}
