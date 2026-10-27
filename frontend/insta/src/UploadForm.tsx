import {
  useMutation,
  useQueryClient,
} from "@tanstack/react-query";
import "./App.css";
import axios from "axios";
import { Button } from "./components/ui/button";
import {
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "./components/ui/dialog";
import {
  FieldDescription,
  FieldGroup,
  FieldLabel,
} from "./components/ui/field";
import { Input } from "./components/ui/input";
import {
  TagsInput,
  TagsInputItem,
  TagsInputLabel,
  TagsInputList,
} from "./components/ui/tags-input";
import { TagsInputInput } from "@diceui/tags-input";

import { useForm } from "@tanstack/react-form";

export function UploadForm() {
  interface UploadFormArgs {
    title: string;
    tags: string[];
    file: File;
  }

  const queryClient = useQueryClient();

  const uploadMutation = useMutation({
    mutationFn: async (value: UploadFormArgs) => {
      await axios.postForm("api/uploads", {
        title: value.title,
        tags: value.tags.join(","),
        file: value.file,
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["feed"] });
    },
  });

  const form = useForm({
    defaultValues: {
      title: "",
      tags: [] as string[],
      file: null as File | null,
    },
    onSubmit: async ({ formApi, value }) => {
      await uploadMutation.mutateAsync(value);
      formApi.reset();
    },
  });

  return (
    <form
      onSubmit={(e) => {
        e.preventDefault();
        e.stopPropagation();
        form.handleSubmit();
      }}
    >
      <DialogHeader>
        <DialogTitle>Upload Image</DialogTitle>
        <DialogDescription>Upload an image with some tags.</DialogDescription>
      </DialogHeader>
      <FieldGroup>
        <form.Field
          name="title"
          validators={{}}
          children={(field) => (
            <>
              <FieldLabel htmlFor={field.name}>Title</FieldLabel>
              <FieldDescription>The title of the image</FieldDescription>
              <Input
                id={field.name}
                name={field.name}
                value={field.state.value}
                placeholder="Title"
                onChange={(e) => field.handleChange(e.target.value)}
              />
            </>
          )}
        />
        <form.Field
          name="tags"
          validators={{}}
          children={(field) => (
            <>
              <FieldLabel>Tags</FieldLabel>
              <FieldDescription>
                add tages to the image allow filtering by and add some flaire
              </FieldDescription>
              <TagsInput
                className="w-full"
                value={field.state.value}
                onValueChange={field.handleChange}
                editable
                addOnPaste
              >
                <TagsInputLabel>Tags</TagsInputLabel>
                <TagsInputList>
                  {field.state.value.map((tag) => (
                    <TagsInputItem key={tag} value={tag}>
                      #{tag}
                    </TagsInputItem>
                  ))}
                </TagsInputList>
                <TagsInputInput placeholder="Add tags..." />
              </TagsInput>
            </>
          )}
        />
        <form.Field
          name="file"
          children={(field) => (
            <div>
              <Input
                type="file"
                onChange={(e) => field.handleChange(e.target.files[0] ?? null)}
              />
              {field.state.value != null && (
                <p>
                  {field.state.value.name} - Size:{" "}
                  {Math.round(field.state.value.size / 1024)}Kb
                </p>
              )}
            </div>
          )}
        />
        <form.Subscribe
          selector={(state) => [state.canSubmit]}
          children={([canSubmit]) => (
            <Button type="submit" disabled={!canSubmit}>
              Submit
            </Button>
          )}
        />
      </FieldGroup>
    </form>
  );
}
