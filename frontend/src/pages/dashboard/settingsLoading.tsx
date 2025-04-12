import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';

export function SettingsSkeleton() {
  return (
    <section className="flex-1 p-0 md:p-4">
      {/* Page title skeleton */}
      <Skeleton className="h-8 w-32 mb-6" />
      
      {/* About Page Card */}
      <Card className="mb-8">
        <CardHeader className="flex flex-row items-center justify-between">
          <CardTitle>
            <Skeleton className="h-6 w-24" />
          </CardTitle>
          <Skeleton className="h-8 w-8 rounded-md" /> {/* Edit button skeleton */}
        </CardHeader>
        <CardContent>
          <div className="space-y-4">
            {/* Title field */}
            <div className="space-y-2">
              <Skeleton className="h-4 w-12" /> {/* Label */}
              <Skeleton className="h-10 w-full" /> {/* Content */}
            </div>
            
            {/* Content field */}
            <div className="space-y-2">
              <Skeleton className="h-4 w-16" /> {/* Label */}
              <Skeleton className="h-24 w-full" /> {/* Content */}
            </div>
            
            {/* Profile Image URL field */}
            <div className="space-y-2">
              <Skeleton className="h-4 w-32" /> {/* Label */}
              <Skeleton className="h-10 w-full" /> {/* Content */}
            </div>
            
            {/* Meta Description field */}
            <div className="space-y-2">
              <Skeleton className="h-4 w-28" /> {/* Label */}
              <Skeleton className="h-16 w-full" /> {/* Content */}
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Contact Page Card */}
      <Card className="mb-8">
        <CardHeader className="flex flex-row items-center justify-between">
          <CardTitle>
            <Skeleton className="h-6 w-28" />
          </CardTitle>
          <Skeleton className="h-8 w-8 rounded-md" /> {/* Edit button skeleton */}
        </CardHeader>
        <CardContent>
          <div className="space-y-4">
            {/* Title field */}
            <div className="space-y-2">
              <Skeleton className="h-4 w-12" /> {/* Label */}
              <Skeleton className="h-10 w-full" /> {/* Content */}
            </div>
            
            {/* Content field */}
            <div className="space-y-2">
              <Skeleton className="h-4 w-16" /> {/* Label */}
              <Skeleton className="h-24 w-full" /> {/* Content */}
            </div>
            
            {/* Email Address field */}
            <div className="space-y-2">
              <Skeleton className="h-4 w-24" /> {/* Label */}
              <Skeleton className="h-10 w-full" /> {/* Content */}
            </div>
            
            {/* Social Links field */}
            <div className="space-y-2">
              <Skeleton className="h-4 w-24" /> {/* Label */}
              <Skeleton className="h-20 w-full" /> {/* Content */}
            </div>
            
            {/* Meta Description field */}
            <div className="space-y-2">
              <Skeleton className="h-4 w-28" /> {/* Label */}
              <Skeleton className="h-16 w-full" /> {/* Content */}
            </div>
          </div>
        </CardContent>
      </Card>
    </section>
  );
}
