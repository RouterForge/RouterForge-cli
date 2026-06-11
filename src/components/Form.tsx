import React, { useState, useEffect, useCallback, createContext, useContext, useMemo } from 'react';

type ValidationRule<T = any> = {
  validate: (value: T) => boolean;
  message: string;
};

type FieldConfig = {
  name: string;
  label: string;
  type?: string;
  rules?: ValidationRule[];
  placeholder?: string;
  options?: { value: string; label: string }[];
  multiSelect?: boolean;
};

type FormContextType<T = Record<string, any>> = {
  values: T;
  errors: Record<string, string>;
  touched: Record<string, boolean>;
  isSubmitting: boolean;
  handleChange: (name: string, value: any) => void;
  handleBlur: (name: string) => void;
  handleSubmit: (onSubmit: (values: T) => void | Promise<void>) => (e: React.FormEvent) => Promise<void>;
  resetForm: () => void;
  setFieldValue: (name: string, value: any) => void;
  getFieldProps: (name: string) => any;
};

const FormContext = createContext<FormContextType | undefined>(undefined);

export const useFormContext = () => {
  const context = useContext(FormContext);
  if (!context) {
    throw new Error('useFormContext must be used within a FormProvider');
  }
  return context;
};

type FormProps<T> = {
  children: React.ReactNode;
  initialValues: T;
  validationRules?: Record<string, ValidationRule[]>;
  onSubmit?: (values: T) => void | Promise<void>;
};

export function Form<T extends Record<string, any>>({
  children,
  initialValues,
  validationRules = {},
  onSubmit,
}: FormProps<T>) {
  const [values, setValues] = useState<T>(initialValues);
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [touched, setTouched] = useState<Record<string, boolean>>({});
  const [isSubmitting, setIsSubmitting] = useState(false);

  const validateField = useCallback(
    (name: string, value: any): string => {
      const rules = validationRules[name] || [];
      for (const rule of rules) {
        if (!rule.validate(value)) {
          return rule.message;
        }
      }
      return '';
    },
    [validationRules]
  );

  const validateForm = useCallback((): boolean => {
    const newErrors: Record<string, string> = {};
    let isValid = true;

    Object.keys(validationRules).forEach((name) => {
      const error = validateField(name, values[name]);
      if (error) {
        newErrors[name] = error;
        isValid = false;
      }
    });

    setErrors(newErrors);
    return isValid;
  }, [validateField, validationRules, values]);

  useEffect(() => {
    // Validate on value changes for touched fields
    const newErrors: Record<string, string> = { ...errors };
    let hasChanges = false;

    Object.keys(touched).forEach((name) => {
      if (touched[name] && validationRules[name]) {
        const error = validateField(name, values[name]);
        if (error !== newErrors[name]) {
          newErrors[name] = error;
          hasChanges = true;
        }
      }
    });

    if (hasChanges) {
      setErrors(newErrors);
    }
  }, [values, touched, validateField, validationRules]);

  const handleChange = useCallback((name: string, value: any) => {
    setValues((prev) => ({ ...prev, [name]: value }));
  }, []);

  const handleBlur = useCallback((name: string) => {
    setTouched((prev) => ({ ...prev, [name]: true }));
  }, []);

  const handleSubmit = useCallback(
    (onSubmitCallback: (values: T) => void | Promise<void>) => {
      return async (e: React.FormEvent) => {
        e.preventDefault();
        
        // Mark all fields as touched
        const allTouched = Object.keys(values).reduce((acc, key) => ({
          ...acc,
          [key]: true,
        }), {});
        setTouched(allTouched);
        
        const isValid = validateForm();
        if (!isValid) return;
        
        setIsSubmitting(true);
        try {
          await onSubmitCallback(values);
        } finally {
          setIsSubmitting(false);
        }
      };
    },
    [values, validateForm]
  );

  const resetForm = useCallback(() => {
    setValues(initialValues);
    setErrors({});
    setTouched({});
    setIsSubmitting(false);
  }, [initialValues]);

  const setFieldValue = useCallback((name: string, value: any) => {
    setValues((prev) => ({ ...prev, [name]: value }));
  }, []);

  const getFieldProps = useCallback(
    (name: string) => ({
      name,
      value: values[name] || '',
      onChange: (e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement | HTMLTextAreaElement>) => {
        const value = e.target.type === 'checkbox' ? e.target.checked : e.target.value;
        handleChange(name, value);
      },
      onBlur: () => handleBlur(name),
      'aria-invalid': !!errors[name],
      'aria-describedby': errors[name] ? `${name}-error` : undefined,
    }),
    [values, errors, handleChange, handleBlur]
  );

  const contextValue = useMemo(
    () => ({
      values,
      errors,
      touched,
      isSubmitting,
      handleChange,
      handleBlur,
      handleSubmit,
      resetForm,
      setFieldValue,
      getFieldProps,
    }),
    [
      values,
      errors,
      touched,
      isSubmitting,
      handleChange,
      handleBlur,
      handleSubmit,
      resetForm,
      setFieldValue,
      getFieldProps,
    ]
  );

  return (
    <FormContext.Provider value={contextValue}>
      <form noValidate>{children}</form>
    </FormContext.Provider>
  );
}

export const withForm = <T extends Record<string, any>>(
  Component: React.ComponentType,
  formProps: Omit<FormProps<T>, 'children'>
) => {
  return function WrappedComponent(props: any) {
    return (
      <Form<T> {...formProps}>
        <Component {...props} />
      </Form>
    );
  };
};

// Export validation rules factory
export const validations = {
  required: (message?: string): ValidationRule => ({
    validate: (value) => {
      if (value === null || value === undefined || value === '') {
        return false;
      }
      if (Array.isArray(value) && value.length === 0) {
        return false;
      }
      return true;
    },
    message: message || 'This field is required',
  }),
  
  email: (message?: string): ValidationRule => ({
    validate: (value) => {
      const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
      return emailRegex.test(value);
    },
    message: message || 'Please enter a valid email address',
  }),
  
  minLength: (min: number, message?: string): ValidationRule => ({
    validate: (value) => value.length >= min,
    message: message || `Must be at least ${min} characters`,
  }),
  
  maxLength: (max: number, message?: string): ValidationRule => ({
    validate: (value) => value.length <= max,
    message: message || `Must be no more than ${max} characters`,
  }),
  
  pattern: (regex: RegExp, message?: string): ValidationRule => ({
    validate: (value) => regex.test(value),
    message: message || 'Invalid format',
  }),
  
  matches: (fieldName: string, message?: string): ValidationRule => ({
    validate: (value, formValues) => value === formValues[fieldName],
    message: message || 'Fields do not match',
  }),
  
  numeric: (message?: string): ValidationRule => ({
    validate: (value) => !isNaN(Number(value)),
    message: message || 'Must be a number',
  }),
  
  minValue: (min: number, message?: string): ValidationRule => ({
    validate: (value) => Number(value) >= min,
    message: message || `Must be at least ${min}`,
  }),
  
  maxValue: (max: number, message?: string): ValidationRule => ({
    validate: (value) => Number(value) <= max,
    message: message || `Must be no more than ${max}`,
  }),
};

// Form field components
export const FormInput: React.FC<
  FieldConfig & { type?: string } & Omit<React.InputHTMLAttributes<HTMLInputElement>, 'onChange' | 'onBlur'>
> = ({ name, label, type = 'text', rules, ...props }) => {
  const { getFieldProps, errors, touched } = useFormContext();
  const fieldProps = getFieldProps(name);
  
  return (
    <div className="form-field">
      <label htmlFor={name} className="form-label">
        {label}
      </label>
      <input
        type={type}
        id={name}
        className={`form-input ${errors[name] && touched[name] ? 'error' : ''}`}
        {...fieldProps}
        {...props}
      />
      {errors[name] && touched[name] && (
        <span id={`${name}-error`} className="form-error" role="alert">
          {errors[name]}
        </span>
      )}
    </div>
  );
};

export const FormTextarea: React.FC<
  FieldConfig & React.TextareaHTMLAttributes<HTMLTextAreaElement>
> = ({ name, label, rules, ...props }) => {
  const { getFieldProps, errors, touched } = useFormContext();
  const fieldProps = getFieldProps(name);
  
  return (
    <div className="form-field">
      <label htmlFor={name} className="form-label">
        {label}
      </label>
      <textarea
        id={name}
        className={`form-textarea ${errors[name] && touched[name] ? 'error' : ''}`}
        {...fieldProps}
        {...props}
      />
      {errors[name] && touched[name] && (
        <span id={`${name}-error`} className="form-error" role="alert">
          {errors[name]}
        </span>
      )}
    </div>
  );
};

export const FormSelect: React.FC<
  FieldConfig & React.SelectHTMLAttributes<HTMLSelectElement>
> = ({ name, label, options = [], rules, ...props }) => {
  const { getFieldProps, errors, touched } = useFormContext();
  const fieldProps = getFieldProps(name);
  
  return (
    <div className="form-field">
      <label htmlFor={name} className="form-label">
        {label}
      </label>
      <select
        id={name}
        className={`form-select ${errors[name] && touched[name] ? 'error' : ''}`}
        {...fieldProps}
        {...props}
      >
        <option value="">Select an option</option>
        {options.map((option) => (
          <option key={option.value} value={option.value}>
            {option.label}
          </option>
        ))}
      </select>
      {errors[name] && touched[name] && (
        <span id={`${name}-error`} className="form-error" role="alert">
          {errors[name]}
        </span>
      )}
    </div>
  );
};

export const FormCheckbox: React.FC<
  FieldConfig & React.InputHTMLAttributes<HTMLInputElement>
> = ({ name, label, rules, ...props }) => {
  const { getFieldProps, errors, touched } = useFormContext();
  const fieldProps = getFieldProps(name);
  
  return (
    <div className="form-field form-field-checkbox">
      <input
        type="checkbox"
        id={name}
        className={`form-checkbox ${errors[name] && touched[name] ? 'error' : ''}`}
        {...fieldProps}
        {...props}
      />
      <label htmlFor={name} className="form-label">
        {label}
      </label>
      {errors[name] && touched[name] && (
        <span id={`${name}-error`} className="form-error" role="alert">
          {errors[name]}
        </span>
      )}
    </div>
  );
};

export const FormError: React.FC<{ name: string }> = ({ name }) => {
  const { errors, touched } = useFormContext();
  
  if (!errors[name] || !touched[name]) {
    return null;
  }
  
  return (
    <span id={`${name}-error`} className="form-error" role="alert">
      {errors[name]}
    </span>
  );
};