import React from 'react';
import { useForm, useField, splitFormProps } from "react-form";
import { useDispatch } from "react-redux";
import { setStreamSettings } from "../redux/actions";

async function sendToFakeServer(values) {
    await new Promise(resolve => setTimeout(resolve, 1000));
    return values;
}

const validate = function(value) {
    if (!value) {
        return "required";
    }
    return false;
}

const InputField = React.forwardRef((props, ref) => {
    // Let's use splitFormProps to get form-specific props
    const [field, fieldOptions, rest] = splitFormProps(props);

    // Use the useField hook with a field and field options
    // to access field state
    const {
        meta: { error, isTouched, isValidating },
        getInputProps
    } = useField(field, fieldOptions);

    // Build the field
    return (
        <>
            <input {...getInputProps({ ref, ...rest })} />{" "}
            {isValidating ? (
                <em>Validating...</em>
            ) : isTouched && error ? (
                <em>{error}</em>
            ) : null}
        </>
    );
});

const Select = React.forwardRef((props, ref) => {
    // Let's use splitFormProps to get form-specific props
    const [field, fieldOptions, rest] = splitFormProps(props);

    // Use the useField hook with a field and field options
    // to access field state
    const {
        meta: { error, isTouched, isValidating },
        getInputProps
    } = useField(field, fieldOptions);

    // Build the field
    return (
        <>
            <select {...getInputProps({ ref, ...rest })} />{" "}
            {isValidating ? (
                <em>Validating...</em>
            ) : isTouched && error ? (
                <em>{error}</em>
            ) : null}
        </>
    );
});

const defaults = {}


// URL-safe base64 encode
const encode64 = (buf) => {
    return btoa(new Uint8Array(buf).reduce((s, b) => s + String.fromCharCode(b), ''))
        .replace(/\+/g, '-') // Convert '+' to '-'
        .replace(/\//g, '_'); // Convert '/' to '_';
}

function generateKey(fieldPath, setFieldValue) {
    const secret = encode64(crypto.getRandomValues(new Uint8Array(12)));
    console.log("set", fieldPath, secret);
    setFieldValue(fieldPath, secret);
    // setValues((prev) => {
    //     return Object.assign({}, prev, {
    //         [key]: secret,
    //     })
    // })
}

function SettingsForm() {
    const dispatch = useDispatch();
    const {
        Form,
        meta: { isSubmitting, canSubmit },
        setFieldValue
    } = useForm({
        defaultValues: defaults,
        onSubmit: async (values, instance) => {
            console.log("submit", values);
            dispatch(setStreamSettings(values))
        },
        debugForm: true
    });

    return (
        <Form>
            <div className="row responsive-label">
                <div className="col-sm-12 col-md-3">
                    <label htmlFor="slug">Slug</label>
                </div>
                <div className="col-sm-12 col-md">
                    <InputField type="text" id="slug" field="slug" defaultValue="" style={{ width: "85%" }} validate={validate} />
                </div>
            </div>
            <div className="row responsive-label">
                <div className="col-sm-12 col-md-3">
                    <label htmlFor="ingestType">Ingest Type</label>
                </div>
                <div className="col-sm-12 col-md">
                    <Select type="text" id="ingestType" field="ingestType" defaultValue="ingest" style={{ width: "85%" }} validate={validate}>
                        <option value="ingest">ingest</option>
                        <option value="relay">relay</option>
                    </Select>
                </div>
            </div>
            <div className="row responsive-label">
                <div className="col-sm-12 col-md-3">
                    <label htmlFor="secret">Auth Key</label>
                </div>
                <div className="col-sm-12 col-md">
                    <InputField type="text" size="3" id="secret" field="secret" defaultValue="" placeholder="no auth" style={{ width: "50%" }} />
                    <button className="secondary generateKey inputAddon" style={{ width: "34%" }} onClick={() => generateKey("secret", setFieldValue)}>Generate key</button>
                </div>
            </div>
            <div className="row responsive-label">
                <div className="col-sm-12 col-md-3">
                    <label htmlFor="notes">Notes</label>
                </div>
                <div className="col-sm-12 col-md">
                    <InputField type="text" size="5" id="notes" field="notes" defaultValue="" placeholder="optional notes" style={{ width: "85%" }} />
                </div>
            </div>
            <div className="row">
                <div className="col-sm-12 col-md-3"></div>
                <div className="col-sm-12 col-md">
                    <button className="primary" type="submit" disabled={!canSubmit}>Submit</button>
                    <em>{isSubmitting ? "Submitting..." : null}</em>
                </div>
            </div>
        </Form>
    )
}

export default SettingsForm;


function AddForm() {
    // Slug       string        `json:"slug"`       // stream slug
    // IngestType string        `json:"ingestType"` // mode: ingest vs. relay
    // Secret     string        `json:"secret"`     // stream secret for authentication
    // Public     bool          `json:"public"`     // whether the stream should be available publically
    // Options    StreamOptions `json:"options"`    // additional stream options

    return
}
